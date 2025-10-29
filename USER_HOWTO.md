# OADP-VMDP User Guide

This guide will help you create backups of your VM files and restore them when needed.

## Prerequisites

Your administrator should have provided you with:
- S3 bucket name
- S3 endpoint URL
- Access key ID
- Secret access key

## First-Time Setup: Connect to Your Backup Repository

Before creating backups, you need to connect to your backup repository. Run this command once:

```bash
oadp-vmdp bslserver create s3 \
  --bucket=YOUR_BUCKET_NAME \
  --endpoint=YOUR_S3_ENDPOINT \
  --access-key=YOUR_ACCESS_KEY \
  --secret-access-key=YOUR_SECRET_KEY
```

You'll be prompted to set a password. **Remember this password** - you'll need it to restore your backups.

## Creating a Backup

To backup a folder or file:

```bash
oadp-vmdp backup create /path/to/folder
```

**Examples:**
```bash
# Backup your home directory
oadp-vmdp backup create /home/myuser

# Backup specific folders
oadp-vmdp backup create /home/myuser/documents /home/myuser/photos

# Backup a single file
oadp-vmdp backup create /home/myuser/important-file.txt
```

## Listing Your Backups

To see all your backups:

```bash
oadp-vmdp backup list
```

To see backups of a specific folder:

```bash
oadp-vmdp backup list /path/to/folder
```

## Restoring Files from a Backup

To restore files to their original location:

```bash
oadp-vmdp backup restore /path/to/folder
```

To restore to a different location:

```bash
oadp-vmdp restore /path/to/folder --target=/path/to/restore/location
```

**Examples:**
```bash
# Restore your documents folder
oadp-vmdp backup restore /home/myuser/documents

# Restore to a different location
oadp-vmdp restore /home/myuser/documents --target=/tmp/restored-docs

# Restore a specific file
oadp-vmdp backup restore /home/myuser/important-file.txt
```

## Connecting to an Existing Repository

If you already created a repository and need to reconnect (e.g., after a reboot):

```bash
oadp-vmdp bslserver connect s3 \
  --bucket=YOUR_BUCKET_NAME \
  --endpoint=YOUR_S3_ENDPOINT \
  --access-key=YOUR_ACCESS_KEY \
  --secret-access-key=YOUR_SECRET_KEY
```

Enter the password you set during repository creation.

## Disconnecting from Repository

When you're done:

```bash
oadp-vmdp bslserver disconnect
```

## Common Tips

- **Backup regularly**: Schedule regular backups of your important folders
- **Test your restores**: Occasionally test restoring to make sure your backups work
- **Keep your password safe**: Without it, you cannot restore your backups
- **Check backup status**: Use `oadp-vmdp bslserver status` to verify your connection

## Need Help?

For more options and advanced features:
```bash
oadp-vmdp --help
oadp-vmdp backup --help
oadp-vmdp restore --help
```
