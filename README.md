# SQLite S3 Backups

A simple Go application to backup your SQLite database to S3 via a cron schedule or on-demand.

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template/I4zGrH)

## Features

- Automated SQLite backups using `VACUUM INTO` command
- **URL-based database access** - Download databases from HTTP/HTTPS or S3 URLs (perfect for Railway!)
- S3-compatible storage support (AWS S3, Cloudflare R2, Backblaze B2, MinIO, etc.)
- Configurable cron scheduling
- Single-shot mode for platform-native cron schedulers
- Optional object lock support
- Lightweight Go binary with minimal dependencies

## Configuration

### Required Environment Variables

- `DATABASE_PATH` - The path or URL to the SQLite database to backup. Supports:
  - Local file paths: `/data/myapp.db`
  - HTTP/HTTPS URLs: `https://example.com/database.db`
  - S3 URLs: `s3://bucket-name/path/to/database.db`

- `AWS_ACCESS_KEY_ID` - AWS access key ID.

- `AWS_SECRET_ACCESS_KEY` - AWS secret access key, sometimes also called an application key.

- `AWS_S3_BUCKET` - The name of the bucket that the access key ID and secret access key are authorized to access.

### Optional Environment Variables

- `AWS_S3_REGION` - The name of the region your bucket is located in. Default: `us-east-1`. Set to `auto` if using Cloudflare R2.

- `BACKUP_CRON_SCHEDULE` - The cron schedule to run the backup on. Default: `0 5 * * *` (daily at 5 AM UTC). Uses standard cron syntax.

- `AWS_S3_ENDPOINT` - The S3 custom endpoint you want to use. Applicable for 3rd party S3 services such as Cloudflare R2, Backblaze B2, or MinIO.

- `AWS_S3_FORCE_PATH_STYLE` - Use path style for the endpoint instead of the default subdomain style, useful for MinIO. Default: `false`

- `RUN_ON_STARTUP` - Run a backup on startup of this application then proceed with making backups on the set schedule. Default: `false`

- `BACKUP_FILE_PREFIX` - Add a prefix to the backup file name. Default: `backup`

- `BUCKET_SUBFOLDER` - Define a subfolder to place the backup files in. Example: `backups/sqlite`

- `SINGLE_SHOT_MODE` - Run a single backup on start and exit when completed. Useful with platform-native cron schedulers. Default: `false`

- `SUPPORT_OBJECT_LOCK` - Enables support for buckets with object lock by computing and providing an MD5 hash with the backup file. Note: This is more resource-intensive. Default: `false`

- `GO_VERSION` - Specify a custom Go version to override the default version set in the Dockerfile. Default: `1.21`

## How It Works

1. If `DATABASE_PATH` is a URL, the application downloads the database to a temporary location
2. The application connects to your SQLite database (local or downloaded)
3. It creates a backup using SQLite's `VACUUM INTO` command, which creates a clean, optimized copy of the database
4. The backup file is uploaded to your S3-compatible storage
5. Temporary files (downloaded database and backup) are automatically cleaned up
6. The process repeats according to your cron schedule, or exits if in single-shot mode

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

## Building from Source

```bash
go mod download
CGO_ENABLED=1 go build -o sqlite-s3-backup main.go
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
