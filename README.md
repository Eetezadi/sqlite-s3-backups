# libSQL S3 Backups

A simple, efficient Go application to backup your libSQL/Turso database to S3-compatible storage via cron schedule or on-demand.

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/I4zGrH)

## Features

- **libSQL/Turso databases**: Direct connection to cloud databases using embedded replicas
- **Optimized backups**: Uses libSQL's embedded replica sync for reliable, consistent copies
- **S3-compatible storage**: Works with AWS S3, Cloudflare R2, Backblaze B2, MinIO, and more
- **Flexible scheduling**: Built-in cron scheduler or single-shot mode for platform-native cron
- **Lightweight**: Single Go binary with minimal dependencies
- **Fast and reliable**: No need to download entire database first - syncs directly to local replica

## Quick Start

```bash
# Download and install
wget https://github.com/yourrepo/libsql-s3-backups/releases/latest/download/libsql-s3-backup
chmod +x libsql-s3-backup

# Configure
export DATABASE_PATH=libsql://your-database.turso.io
export DATABASE_AUTH_TOKEN=your-turso-auth-token
export AWS_ACCESS_KEY_ID=your_key_id
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_S3_BUCKET=my-backups

# Validate and run
./libsql-s3-backup --validate  # Check configuration
./libsql-s3-backup             # Start backup service
```

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_PATH` | libSQL database URL (e.g., `libsql://your-db.turso.io` or `https://your-db.turso.io`) |
| `DATABASE_AUTH_TOKEN` | Authentication token for libSQL/Turso database |
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

### Database URL Format

The `DATABASE_PATH` must be a valid libSQL database URL:

**libSQL protocol:**
```bash
DATABASE_PATH=libsql://your-database.turso.io
DATABASE_AUTH_TOKEN=your-turso-auth-token
```

**HTTPS (Turso databases):**
```bash
DATABASE_PATH=https://your-database.turso.io
DATABASE_AUTH_TOKEN=your-turso-auth-token
```

## Examples

### Daily backups at 3 AM UTC

```bash
DATABASE_PATH=libsql://my-database.turso.io
DATABASE_AUTH_TOKEN=your-turso-auth-token
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=my-backups
BACKUP_CRON_SCHEDULE="0 3 * * *"
```

### Production database backups every 6 hours

```bash
DATABASE_PATH=libsql://my-prod-db.turso.io
DATABASE_AUTH_TOKEN=eyJhbGc...your-token
AWS_S3_BUCKET=turso-backups
BACKUP_CRON_SCHEDULE="0 */6 * * *"
BUCKET_SUBFOLDER=production
```

### Cloudflare R2 storage

```bash
DATABASE_PATH=libsql://my-database.turso.io
DATABASE_AUTH_TOKEN=your-token
AWS_S3_ENDPOINT=https://your-account.r2.cloudflarestorage.com
AWS_S3_REGION=auto
AWS_S3_BUCKET=your-bucket
```

### MinIO storage

```bash
DATABASE_PATH=libsql://my-database.turso.io
DATABASE_AUTH_TOKEN=your-token
AWS_S3_ENDPOINT=https://minio.example.com
AWS_S3_FORCE_PATH_STYLE=true
AWS_S3_REGION=us-east-1
```

### Single-shot mode (external cron)

```bash
DATABASE_PATH=libsql://my-database.turso.io
DATABASE_AUTH_TOKEN=your-token
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_S3_BUCKET=my-backups
SINGLE_SHOT_MODE=true  # Run once and exit
```

## Deployment

### Railway

Click the **Deploy on Railway** button above, then configure the environment variables in your Railway dashboard.

### Docker

```bash
docker build -t libsql-s3-backup .

docker run -d \
  -e DATABASE_PATH=libsql://my-database.turso.io \
  -e DATABASE_AUTH_TOKEN=your-token \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  -e AWS_S3_BUCKET=your_bucket \
  libsql-s3-backup
```

### Build from Source

Requires Go 1.21+ and CGO enabled (required by libSQL):

```bash
git clone https://github.com/yourrepo/libsql-s3-backups.git
cd libsql-s3-backups

CGO_ENABLED=1 go build -o libsql-s3-backup .
./libsql-s3-backup --version
```

## Development

```bash
# Install dependencies
go mod download

# Run tests
CGO_ENABLED=1 go test -v ./...

# Build
CGO_ENABLED=1 go build -o libsql-s3-backup .

# Run locally
./libsql-s3-backup --validate  # Check config
./libsql-s3-backup             # Start service
```

## License

MIT
