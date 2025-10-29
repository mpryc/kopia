package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kopia/kopia/internal/atomicfile"
	"github.com/kopia/kopia/internal/ospath"
	"github.com/kopia/kopia/repo/blob"
	"github.com/kopia/kopia/repo/blob/throttling"
	"github.com/kopia/kopia/repo/content"
	"github.com/kopia/kopia/repo/format"
)

const configDirMode = 0o700

const oadpPrefix = "oadp-vmdp/"

// ErrCannotWriteToRepoConnectionWithPermissiveCacheLoading error to indicate.
var ErrCannotWriteToRepoConnectionWithPermissiveCacheLoading = errors.New("cannot write to repo connection with permissive cache loading")

// denormalizeOADPPrefix removes the "oadp-vmdp/" prefix from an S3 connection's prefix if present.
// This is used when saving config to store only the user-provided prefix.
// Uses reflection to avoid import cycles with blob/s3 package.
func denormalizeOADPPrefix(ci *blob.ConnectionInfo) {
	if ci.Type != "s3" || ci.Config == nil {
		return
	}

	// Use reflection to access the Prefix field
	v := reflect.ValueOf(ci.Config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	prefixField := v.FieldByName("Prefix")
	if !prefixField.IsValid() || !prefixField.CanSet() || prefixField.Kind() != reflect.String {
		return
	}

	currentPrefix := prefixField.String()
	prefixField.SetString(strings.TrimPrefix(currentPrefix, oadpPrefix))
}

// normalizeOADPPrefix prepends "oadp-vmdp/" to an S3 connection's prefix if not already present.
// This is used when loading config to apply the runtime prefix normalization.
// Uses reflection to avoid import cycles with blob/s3 package.
func normalizeOADPPrefix(ci *blob.ConnectionInfo) {
	if ci.Type != "s3" || ci.Config == nil {
		return
	}

	// Use reflection to access the Prefix field
	v := reflect.ValueOf(ci.Config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	prefixField := v.FieldByName("Prefix")
	if !prefixField.IsValid() || !prefixField.CanSet() || prefixField.Kind() != reflect.String {
		return
	}

	currentPrefix := prefixField.String()

	// Only add the prefix if it's not already there
	if !strings.HasPrefix(currentPrefix, oadpPrefix) {
		// Remove any leading slashes from user prefix
		cleanedPrefix := strings.TrimLeft(currentPrefix, "/")
		prefixField.SetString(oadpPrefix + cleanedPrefix)
	}
}

// ClientOptions contains client-specific options that are persisted in local configuration file.
type ClientOptions struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`

	ReadOnly               bool `json:"readonly,omitempty"`
	PermissiveCacheLoading bool `json:"permissiveCacheLoading,omitempty"`

	// Description is human-readable description of the repository to use in the UI.
	Description string `json:"description,omitempty"`

	EnableActions bool `json:"enableActions"`

	FormatBlobCacheDuration time.Duration `json:"formatBlobCacheDuration,omitempty"`

	Throttling *throttling.Limits `json:"throttlingLimits,omitempty"`
}

// ApplyDefaults returns a copy of ClientOptions with defaults filled out.
func (o ClientOptions) ApplyDefaults(ctx context.Context, defaultDesc string) ClientOptions {
	if o.Hostname == "" {
		o.Hostname = GetDefaultHostName(ctx)
	}

	if o.Username == "" {
		o.Username = GetDefaultUserName(ctx)
	}

	if o.Description == "" {
		o.Description = defaultDesc
	}

	if o.FormatBlobCacheDuration == 0 {
		o.FormatBlobCacheDuration = format.DefaultRepositoryBlobCacheDuration
	}

	return o
}

// Override returns ClientOptions that overrides fields present in the provided ClientOptions.
func (o ClientOptions) Override(other ClientOptions) ClientOptions {
	if other.Description != "" {
		o.Description = other.Description
	}

	if other.Hostname != "" {
		o.Hostname = other.Hostname
	}

	if other.Username != "" {
		o.Username = other.Username
	}

	if other.ReadOnly {
		o.ReadOnly = other.ReadOnly
	}

	return o
}

// UsernameAtHost returns 'username@hostname' string.
func (o ClientOptions) UsernameAtHost() string {
	return o.Username + "@" + o.Hostname
}

// LocalConfig is a configuration of Kopia stored in a configuration file.
type LocalConfig struct {
	// APIServer is only provided for remote repository.
	APIServer *APIServerInfo `json:"apiServer,omitempty"`

	// Storage is only provided for direct repository access.
	Storage *blob.ConnectionInfo `json:"storage,omitempty"`

	Caching *content.CachingOptions `json:"caching,omitempty"`

	ClientOptions
}

// writeToFile writes the config to a given file.
func (lc *LocalConfig) writeToFile(filename string) error {
	lc2 := *lc

	if lc.Caching != nil {
		lc2.Caching = lc.Caching.CloneOrDefault()

		// try computing relative pathname from config dir to the cache dir.
		d, err := filepath.Rel(filepath.Dir(filename), lc.Caching.CacheDirectory)
		if err == nil {
			lc2.Caching.CacheDirectory = d
		}
	}

	// Denormalize S3 prefix before saving (remove "oadp-vmdp/" prefix)
	if lc2.Storage != nil {
		// Make a copy of the storage config to avoid modifying the original
		storageCopy := *lc2.Storage
		lc2.Storage = &storageCopy

		// If it's S3 and has a Config, make a deep copy of the config struct
		if lc2.Storage.Type == "s3" && lc2.Storage.Config != nil {
			configValue := reflect.ValueOf(lc2.Storage.Config)
			if configValue.Kind() == reflect.Ptr && !configValue.IsNil() {
				// Create a new instance of the same type
				configCopy := reflect.New(configValue.Elem().Type())
				configCopy.Elem().Set(configValue.Elem())
				lc2.Storage.Config = configCopy.Interface()
			}
		}

		denormalizeOADPPrefix(lc2.Storage)
	}

	b, err := json.MarshalIndent(lc2, "", "  ")
	if err != nil {
		return errors.Wrap(err, "error creating config file contents")
	}

	if err = os.MkdirAll(filepath.Dir(filename), configDirMode); err != nil {
		return errors.Wrap(err, "unable to create config directory")
	}

	return errors.Wrap(atomicfile.Write(filename, bytes.NewReader(b)), "error writing file")
}

// LoadConfigFromFile reads the local configuration from the specified file.
func LoadConfigFromFile(fileName string) (*LocalConfig, error) {
	f, err := os.Open(fileName) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "error loading config file")
	}
	defer f.Close() //nolint:errcheck

	var lc LocalConfig

	if err := json.NewDecoder(f).Decode(&lc); err != nil {
		return nil, errors.Wrap(err, "error decoding config json")
	}

	// cache directory is stored as relative to config file name, resolve it to absolute.
	if lc.Caching != nil {
		if lc.Caching.CacheDirectory != "" && !ospath.IsAbs(lc.Caching.CacheDirectory) {
			lc.Caching.CacheDirectory = filepath.Join(filepath.Dir(fileName), lc.Caching.CacheDirectory)
		}

		// override cache directory from the environment variable.
		if cd := os.Getenv("OADP_CACHE_DIRECTORY"); cd != "" && ospath.IsAbs(cd) {
			lc.Caching.CacheDirectory = cd
		}
	}

	if lc.PermissiveCacheLoading && os.Getenv("KOPIA_UPGRADE_LOCK_ENABLED") == "" {
		return nil, errors.New("must have set KOPIA_UPGRADE_LOCK_ENABLED when connecting to repository with permissive cache loading")
	}

	// Normalize S3 prefix after loading (add "oadp-vmdp/" prefix at runtime)
	if lc.Storage != nil {
		normalizeOADPPrefix(lc.Storage)
	}

	return &lc, nil
}
