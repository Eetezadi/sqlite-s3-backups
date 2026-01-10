# SQLite S3 Backups

A simple, efficient Go application to backup your SQLite or libSQL database to S3-compatible storage via cron schedule or on-demand.

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/I4zGrH)

## Features

- **Multiple database sources**: Local files, HTTP/HTTPS URLs, S3 URLs, or libSQL/Turso databases
- **Optimized backups**: Uses SQLite's `VACUUM INTO` for clean, defragmented copies
- **S3-compatible storage**: Works with AWS S3, Cloudflare R2, Backblaze B2, MinIO, and more
- **Flexible scheduling**: Built-in cron scheduler or single-shot mode for platform-native cron
- **Turso/libSQL support**: Direct connection to cloud databases without downloading first
- **Lightweight**: Single Go binary with minimal dependencies

## Quick Start

```bash
# Download and install
wget https://github.com/yourrepo/sqlite-s3-backups/releases/latest/download/sqlite-s3-backup
chmod +x sqlite-s3-backup

# Configure
export DATABASE_PATH=/data/myapp.db
export AWS_ACCESS_KEY_ID=your_key_id
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_S3_BUCKET=my-backups

# Validate and run
./sqlite-s3-backup --validate  # Check configuration
./sqlite-s3-backup             # Start backup service
```

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_PATH` | Path or URL to your database (see Database Sources below) |
| `AWS_ACCESS_KEY_ID` | AWS access key ID |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key |
| `AWS_S3_BUCKET` | S3 bucket name for backups |

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AWS_S3_REGION` | `us-east-1` | S3 region (`auto` for Cloudflare R2) |
| `AWS_S3_ENDPOINT` | - | Custom S3 endpoint (for R2, MinIO, etc.) |
| `AWS_S3_FORCE_PATH_STYLE` | `false` | Use path-style URLs (required for MinIO) |
| `BACKUP_CRON_SCHEDULE` | `0 5 * * *` | Cron schedule (daily at 5 AM UTC) |
| `BACKUP_FILE_PREFIX` | `backup` | Prefix for backup filenames |
| `BUCKET_SUBFOLDER` | - | Subfolder path within bucket |
| `RUN_ON_STARTUP` | `false` | Run backup immediately on startup |
| `SINGLE_SHOT_MODE` | `false` | Run once and exit (for external cron) |
| `SUPPORT_OBJECT_LOCK` | `false` | Enable S3 object lock with MD5 hash |
| `DATABASE_AUTH_TOKEN` | - | Authentication token for libSQL/Turso |

### Database Sources

The `DATABASE_PATH` variable supports multiple source types:

**Local file:**
```bash
DATABASE_PATH=/data/myapp.db
```

**HTTP/HTTPS URL** (perfect for Railway services):
```bash
DATABASE_PATH=https://myapp.railway.app/db/database.db
```

**S3 URL** (download from another bucket):
```bash
DATABASE_PATH=s3://source-bucket/path/database.db
```

**libSQL/Turso database** (direct connection):
```bash
DATABASE_PATH=libsql://your-database.turso.io
# or
DATABASE_PATH=https://your-database.turso.io
DATABASE_AUTH_TOKEN=your-turso-auth-token
```

## Examples

### Daily backups at 3 AM UTC

```bash
DATABASE_PATH=/data/myapp.db
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=my-backups
BACKUP_CRON_SCHEDULE="0 3 * * *"
```

### Turso database backups every 6 hours

```bash
DATABASE_PATH=libsql://my-prod-db.turso.io
DATABASE_AUTH_TOKEN=eyJhbGc...your-token
AWS_S3_BUCKET=turso-backups
BACKUP_CRON_SCHEDULE="0 */6 * * *"
BUCKET_SUBFOLDER=production
```

### Railway: Backup via HTTP endpoint

```bash
DATABASE_PATH=https://myapp.railway.app/db/database.db
AWS_S3_BUCKET=my-backups
BACKUP_CRON_SCHEDULE="0 */12 * * *"  # Every 12 hours
```

Example HTTP endpoint in your app:
```go
http.HandleFunc("/db/database.db", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "/data/database.db")
})
```

### Cloudflare R2 storage

```bash
AWS_S3_ENDPOINT=https://your-account.r2.cloudflarestorage.com
AWS_S3_REGION=auto
AWS_S3_BUCKET=your-bucket
```

### MinIO storage

```bash
AWS_S3_ENDPOINT=https://minio.example.com
AWS_S3_FORCE_PATH_STYLE=true
AWS_S3_REGION=us-east-1
```

### Single-shot mode (external cron)

```bash
SINGLE_SHOT_MODE=true  # Run once and exit
```

## Deployment

### Railway

Click the **Deploy on Railway** button above, then configure the environment variables in your Railway dashboard.

For Railway deployments without shared volumes, use the HTTP URL method shown in the examples above.

### Docker

```bash
docker build -t sqlite-s3-backup .

docker run -d \
  -e DATABASE_PATH=/data/mydb.db \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_S3_BUCKET=your_bucket \
  -v /path/to/database:/data \
  sqlite-s3-backup
```

### Build from Source

Requires Go 1.21+ and CGO enabled:

```bash
git clone https://github.com/yourrepo/sqlite-s3-backups.git
cd sqlite-s3-backups

CGO_ENABLED=1 go build -o sqlite-s3-backup .
./sqlite-s3-backup --version
```

## Development

```bash
# Install dependencies
go mod download

# Run tests
CGO_ENABLED=1 go test -v ./...

# Build
CGO_ENABLED=1 go build -o sqlite-s3-backup .

# Run locally
./sqlite-s3-backup --validate  # Check config
./sqlite-s3-backup             # Start service
```

## License

MIT
