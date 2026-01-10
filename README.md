# SQLite S3 Backups

A simple, efficient Go application to backup your SQLite or libSQL database to S3 via cron schedule or on-demand.

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/I4zGrH)

## Quick Start

```bash
# Set required environment variables
export DATABASE_PATH=/data/myapp.db
export AWS_ACCESS_KEY_ID=your_key_id
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_S3_BUCKET=my-backups

# Run a single backup
./sqlite-s3-backup --validate  # Check configuration
./sqlite-s3-backup             # Run backup service
```

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Required Variables](#required-environment-variables)
  - [Optional Variables](#optional-environment-variables)
- [How It Works](#how-it-works)
- [Database Sources](#database-sources)
  - [SQLite VACUUM INTO](#sqlite-vacuum-into)
  - [URL-Based Access](#url-based-database-access)
  - [libSQL / Turso Support](#libsql--turso-database-support)
- [Deployment](#deployment)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## Features

- Automated SQLite/libSQL backups using `VACUUM INTO` command
- **libSQL/Turso support** - Direct connection to libSQL databases via `libsql://` URLs
- **URL-based database access** - Download databases from HTTP/HTTPS or S3 URLs (perfect for Railway!)
- S3-compatible storage support (AWS S3, Cloudflare R2, Backblaze B2, MinIO, etc.)
- Configurable cron scheduling
- Single-shot mode for platform-native cron schedulers
- Optional object lock support
- Lightweight Go binary with minimal dependencies

## Installation

### Download Pre-built Binary

```bash
# Download latest release (replace with actual release URL)
wget https://github.com/yourrepo/sqlite-s3-backups/releases/latest/download/sqlite-s3-backup
chmod +x sqlite-s3-backup
./sqlite-s3-backup --version
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourrepo/sqlite-s3-backups.git
cd sqlite-s3-backups

# Build (requires Go 1.21+ and CGO)
CGO_ENABLED=1 go build -o sqlite-s3-backup .

# Verify
./sqlite-s3-backup --version
```

### Docker

```bash
# Build
docker build -t sqlite-s3-backup .

# Run
docker run -e DATABASE_PATH=/data/mydb.db \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_S3_BUCKET=your_bucket \
  -v /path/to/your/database:/data \
  sqlite-s3-backup
```

## Configuration

### Required Environment Variables

- `DATABASE_PATH` - The path or URL to the SQLite/libSQL database to backup. Supports:
  - Local file paths: `/data/myapp.db`
  - HTTP/HTTPS URLs: `https://example.com/database.db`
  - S3 URLs: `s3://bucket-name/path/to/database.db`
  - libSQL URLs: `libsql://your-database.turso.io` or `https://your-database.turso.io`

- `AWS_ACCESS_KEY_ID` - AWS access key ID.

- `AWS_SECRET_ACCESS_KEY` - AWS secret access key, sometimes also called an application key.

- `AWS_S3_BUCKET` - The name of the bucket that the access key ID and secret access key are authorized to access.

### Optional Environment Variables

- `AWS_S3_REGION` - The name of the region your bucket is located in. Default: `us-east-1`. Set to `auto` if using Cloudflare R2.

- `BACKUP_CRON_SCHEDULE` - The cron schedule to run the backup on. Default: `0 5 * * *` (daily at 5 AM UTC). Uses standard cron syntax.

- `AWS_S3_ENDPOINT` - The S3 custom endpoint you want to use. Applicable for 3rd party S3 services such as Cloudflare R2, Backblaze B2, or MinIO.

- `AWS_S3_FORCE_PATH_STYLE` - Use path style for the endpoint instead of the default subdomain style, useful for MinIO. Default: `false`

- `DATABASE_AUTH_TOKEN` - Authentication token for libSQL/Turso databases. Required when using `libsql://` URLs. Get this from your Turso database dashboard.

- `RUN_ON_STARTUP` - Run a backup on startup of this application then proceed with making backups on the set schedule. Default: `false`

- `BACKUP_FILE_PREFIX` - Add a prefix to the backup file name. Default: `backup`

- `BUCKET_SUBFOLDER` - Define a subfolder to place the backup files in. Example: `backups/sqlite`

- `SINGLE_SHOT_MODE` - Run a single backup on start and exit when completed. Useful with platform-native cron schedulers. Default: `false`

- `SUPPORT_OBJECT_LOCK` - Enables support for buckets with object lock by computing and providing an MD5 hash with the backup file. Note: This is more resource-intensive. Default: `false`

- `GO_VERSION` - Specify a custom Go version to override the default version set in the Dockerfile. Default: `1.21`

## How It Works

1. **For libSQL/Turso databases**: Connects directly to the database server using the libSQL protocol
2. **For HTTP/S3 URLs**: Downloads the database file to a temporary location
3. **For local files**: Uses the file path directly
4. Creates a backup using `VACUUM INTO` command, which creates a clean, optimized copy
5. Uploads the backup file to your S3-compatible storage
6. Cleans up temporary files
7. Repeats according to your cron schedule, or exits if in single-shot mode

## SQLite VACUUM INTO

This application uses SQLite's `VACUUM INTO` command, which:
- Creates a fresh copy of the database without fragmentation
- Removes deleted data and optimizes the database structure
- Produces a clean backup file
- Does not require external tools like `sqlite3` CLI
- Works with the database while it's in use (with proper locking)

## URL-Based Database Access

Perfect for cloud platforms like Railway where services can't share volumes! The application can download your database from a URL before backing it up.

### Supported URL Schemes

1. **HTTP/HTTPS** - Download from any web server
   ```bash
   DATABASE_PATH=https://myapp.example.com/database.db
   ```
   Use this when your application exposes the database file via HTTP (e.g., using a simple file server)

2. **S3** - Download directly from S3 or S3-compatible storage
   ```bash
   DATABASE_PATH=s3://my-bucket/databases/production.db
   ```
   Use this when your application writes the database to S3

3. **Local File** - Traditional file path (backwards compatible)
   ```bash
   DATABASE_PATH=/data/myapp.db
   ```

### Railway Deployment Pattern

For Railway deployments with separate services:

**Option 1: HTTP Endpoint**
- Your app service runs a simple file server that serves the database file
- This backup service downloads via HTTP URL
- Example: `DATABASE_PATH=https://myapp.railway.app/db/database.db`

**Option 2: Shared S3 Storage**
- Your app service periodically uploads database to S3
- This backup service downloads from S3, backs it up, and uploads to backup location
- Example: `DATABASE_PATH=s3://my-app-bucket/live/database.db`

### Example: Simple HTTP File Server in Your App

Add this to your application to expose the database:

```go
// Serve database file at /db/database.db
http.HandleFunc("/db/database.db", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "/data/database.db")
})
```

Then set: `DATABASE_PATH=https://your-app-url.railway.app/db/database.db`

## libSQL / Turso Database Support

This application supports direct backup of libSQL and Turso databases without downloading the entire database first. This is perfect for cloud-native applications using Turso.

### How It Works

Instead of downloading the database file, the application:
1. Connects directly to your libSQL/Turso database using the libSQL protocol
2. Executes `VACUUM INTO` on the server to create an optimized backup
3. Downloads only the resulting backup file
4. Uploads the backup to S3

This is more efficient than downloading the entire database, backing it up locally, and uploading again.

### Configuration

```bash
# Turso database URL
DATABASE_PATH=libsql://your-database.turso.io

# Or use HTTPS URL (automatically detected as Turso)
DATABASE_PATH=https://your-database.turso.io

# Authentication token from Turso dashboard
DATABASE_AUTH_TOKEN=your-turso-auth-token

# S3 configuration
AWS_ACCESS_KEY_ID=your-s3-key
AWS_SECRET_ACCESS_KEY=your-s3-secret
AWS_S3_BUCKET=my-backups
```

### Getting Your Turso Credentials

1. Create a database at [turso.tech](https://turso.tech)
2. Get your database URL: `turso db show <database-name>`
3. Create an auth token: `turso db tokens create <database-name>`
4. Use these in your environment variables

### Example: Daily Turso Backups

```bash
DATABASE_PATH=libsql://my-app-production.turso.io
DATABASE_AUTH_TOKEN=eyJhbGc...your-token
AWS_S3_BUCKET=my-turso-backups
BACKUP_CRON_SCHEDULE="0 2 * * *"  # Daily at 2 AM
BUCKET_SUBFOLDER=production
```

## Deployment

### Docker

```bash
docker build -t sqlite-s3-backup .
docker run -e DATABASE_PATH=/data/mydb.db \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_S3_BUCKET=your_bucket \
  -v /path/to/your/database:/data \
  sqlite-s3-backup
```

### Railway

Click the "Deploy on Railway" button above and configure the required environment variables.

### Platform-Native Cron

For platforms with built-in cron schedulers, use `SINGLE_SHOT_MODE=true`:

```bash
# This will run one backup and exit
SINGLE_SHOT_MODE=true ./sqlite-s3-backup
```

## Examples

### Daily backups at 3 AM UTC

```bash
BACKUP_CRON_SCHEDULE="0 3 * * *"
RUN_ON_STARTUP=false
```

### Backup on startup and every 6 hours

```bash
BACKUP_CRON_SCHEDULE="0 */6 * * *"
RUN_ON_STARTUP=true
```

### Single backup for cron jobs

```bash
SINGLE_SHOT_MODE=true
RUN_ON_STARTUP=false  # Not needed, will be ignored
```

### Railway: Backup from HTTP URL

```bash
DATABASE_PATH=https://myapp.railway.app/db/database.db
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=my-backups
BACKUP_CRON_SCHEDULE="0 */12 * * *"  # Every 12 hours
```

### Railway: Backup from S3 source

```bash
DATABASE_PATH=s3://my-app-storage/live/database.db
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=my-backups
BUCKET_SUBFOLDER=backups  # Store backups in a subfolder
```

### Turso: Direct database backup

```bash
DATABASE_PATH=libsql://my-prod-db.turso.io
DATABASE_AUTH_TOKEN=eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9...
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=turso-backups
BACKUP_CRON_SCHEDULE="0 0 * * *"  # Daily at midnight
```

### Using with Cloudflare R2

```bash
AWS_S3_ENDPOINT=https://your-account.r2.cloudflarestorage.com
AWS_S3_REGION=auto
AWS_S3_BUCKET=your-bucket
```

### Using with MinIO

```bash
AWS_S3_ENDPOINT=https://minio.example.com
AWS_S3_FORCE_PATH_STYLE=true
AWS_S3_REGION=us-east-1
```

## Troubleshooting

### CGO Build Errors

**Problem**: `build constraints exclude all Go files` or `undefined: sqlite3`

**Solution**: Ensure CGO is enabled:
```bash
CGO_ENABLED=1 go build -o sqlite-s3-backup .
```

On Alpine Linux, install build dependencies:
```bash
apk add --no-cache gcc musl-dev sqlite-dev
```

### libSQL Connection Errors

**Problem**: `failed to connect to libSQL database`

**Solutions**:
- Verify your `DATABASE_AUTH_TOKEN` is correct (get it from Turso dashboard)
- Check that the database URL is correct (should be `libsql://your-db.turso.io` or `https://your-db.turso.io`)
- Ensure your auth token hasn't expired

### S3 Upload Fails

**Problem**: `S3 upload failed` or permission errors

**Solutions**:
- Verify `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are correct
- Check bucket permissions (must allow PutObject)
- For Cloudflare R2, ensure `AWS_S3_REGION=auto`
- For MinIO, set `AWS_S3_FORCE_PATH_STYLE=true`
- Check the `AWS_S3_ENDPOINT` format (should include protocol: `https://`)

### Database Download Errors

**Problem**: `failed to download database` from URL

**Solutions**:
- **HTTP/HTTPS**: Verify the URL is accessible and returns the database file
- **S3 URLs**: Ensure your AWS credentials have GetObject permission for the source bucket
- Check network connectivity to the source
- Verify the URL scheme is correct (`http://`, `https://`, or `s3://`)

### VACUUM INTO Errors

**Problem**: `VACUUM INTO failed` or `database is locked`

**Solutions**:
- Ensure the source database is not exclusively locked by another process
- Check available disk space (VACUUM creates a full copy)
- For libSQL databases, verify the database supports VACUUM INTO (most recent versions do)
- Check SQLite version is 3.27.0+ (when VACUUM INTO was added)

### Object Lock Issues

**Problem**: `MD5 hash required` or object lock errors

**Solution**: Enable object lock support:
```bash
SUPPORT_OBJECT_LOCK=true
```

Note: This computes MD5 hash of the backup file, which uses more memory.

### Cron Schedule Not Working

**Problem**: Backups don't run on schedule

**Solutions**:
- Verify cron syntax is correct (5 fields: minute hour day month weekday)
- Check logs for schedule parsing errors
- Test with a frequent schedule first: `BACKUP_CRON_SCHEDULE="*/5 * * * *"` (every 5 minutes)
- Ensure the application is running continuously (not in `SINGLE_SHOT_MODE`)

### Configuration Validation

To test your configuration without running a backup:
```bash
./sqlite-s3-backup --validate
```

This will verify all required environment variables are set and the cron schedule is valid.

## Building from Source

```bash
go mod download
CGO_ENABLED=1 go build -o sqlite-s3-backup .
```

Note: CGO must be enabled for SQLite support.

## Development

```bash
# Download dependencies
go mod download

# Run locally
go run main.go

# Build
go build -o sqlite-s3-backup main.go

# Run tests (after creating)
go test ./...
```

## License

MIT
