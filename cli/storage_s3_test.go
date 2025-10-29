package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kopia/kopia/internal/testutil"
)

var (
	fakeCertContent         = []byte("fake certificate content")
	fakeCertContentAsBase64 = base64.StdEncoding.EncodeToString(fakeCertContent)
)

func TestLoadPEMBase64(t *testing.T) {
	var s3flags storageS3Flags

	s3flags = storageS3Flags{rootCaPemBase64: ""}
	require.NoError(t, s3flags.preActionLoadPEMBase64(nil))

	s3flags = storageS3Flags{rootCaPemBase64: "AA=="}
	require.NoError(t, s3flags.preActionLoadPEMBase64(nil))

	s3flags = storageS3Flags{rootCaPemBase64: fakeCertContentAsBase64}
	require.NoError(t, s3flags.preActionLoadPEMBase64(nil))
	require.Equal(t, fakeCertContent, s3flags.s3options.RootCA, "content of RootCA should be %v", fakeCertContent)

	s3flags = storageS3Flags{rootCaPemBase64: "!"}
	err := s3flags.preActionLoadPEMBase64(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "illegal base64 data")
}

func TestLoadPEMPath(t *testing.T) {
	var s3flags storageS3Flags

	tempdir := testutil.TempDirectory(t)
	certpath := filepath.Join(tempdir, "certificate-filename")

	require.NoError(t, os.WriteFile(certpath, fakeCertContent, 0o644))

	// Test regular file
	s3flags = storageS3Flags{rootCaPemPath: certpath}
	require.NoError(t, s3flags.preActionLoadPEMPath(nil))
	require.Equal(t, fakeCertContent, s3flags.s3options.RootCA, "content of RootCA should be %v", fakeCertContent)

	// Test inexistent file
	s3flags = storageS3Flags{rootCaPemPath: "/does-not-exists"}
	err := s3flags.preActionLoadPEMPath(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error opening root-ca-pem-path")
}

func TestLoadPEMBoth(t *testing.T) {
	s3flags := storageS3Flags{rootCaPemBase64: "AA==", rootCaPemPath: "/tmp/blah"}
	require.NoError(t, s3flags.preActionLoadPEMBase64(nil))
	err := s3flags.preActionLoadPEMPath(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestNormalizeOADPPrefix(t *testing.T) {
	tests := []struct {
		name        string
		userPrefix  string
		expected    string
		expectError bool
	}{
		{
			name:        "prefix with trailing slash",
			userPrefix:  "abc/",
			expected:    "oadp-vmdp/abc/",
			expectError: false,
		},
		{
			name:        "prefix without trailing slash",
			userPrefix:  "abc",
			expected:    "oadp-vmdp/abc",
			expectError: false,
		},
		{
			name:        "prefix with leading and trailing slash",
			userPrefix:  "/abc/bce/",
			expected:    "oadp-vmdp/abc/bce/",
			expectError: false,
		},
		{
			name:        "prefix with leading slash no trailing slash",
			userPrefix:  "/abc/bce",
			expected:    "oadp-vmdp/abc/bce",
			expectError: false,
		},
		{
			name:        "just slash",
			userPrefix:  "/",
			expected:    "oadp-vmdp/",
			expectError: false,
		},
		{
			name:        "empty string",
			userPrefix:  "",
			expected:    "oadp-vmdp/",
			expectError: false,
		},
		{
			name:        "multiple leading slashes",
			userPrefix:  "///abc/",
			expected:    "oadp-vmdp/abc/",
			expectError: false,
		},
		// Error cases: user provides oadp-vmdp in prefix
		{
			name:        "user provides exact oadp-vmdp prefix",
			userPrefix:  "oadp-vmdp/",
			expected:    "",
			expectError: true,
		},
		{
			name:        "user provides oadp-vmdp with leading slash",
			userPrefix:  "/oadp-vmdp/",
			expected:    "",
			expectError: true,
		},
		{
			name:        "user provides oadp-vmdp in middle",
			userPrefix:  "some/oadp-vmdp/path",
			expected:    "",
			expectError: true,
		},
		{
			name:        "user provides uppercase OADP-VMDP",
			userPrefix:  "OADP-VMDP/",
			expected:    "",
			expectError: true,
		},
		{
			name:        "user provides mixed case OaDp-VmDp",
			userPrefix:  "OaDp-VmDp/backup",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeOADPPrefix(tt.userPrefix)
			if tt.expectError {
				require.Error(t, err, "normalizeOADPPrefix(%q) should return an error", tt.userPrefix)
				require.Contains(t, err.Error(), "must not contain 'oadp-vmdp'")
			} else {
				require.NoError(t, err, "normalizeOADPPrefix(%q) should not return an error", tt.userPrefix)
				require.Equal(t, tt.expected, result, "normalizeOADPPrefix(%q) should return %q", tt.userPrefix, tt.expected)
			}
		})
	}
}
