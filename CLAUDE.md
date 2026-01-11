# libSQL S3 Backups - Project Documentation

## Project Overview

This is a Go application that creates automated backups of libSQL/Turso databases and uploads them to S3-compatible storage. The project was refactored from a dual SQLite/libSQL backup tool to be **libSQL-only**.

## Architecture

### Core Components

1. **Backup System** ([backup.go](backup.go))
   - Uses libSQL's embedded replica feature for backups
   - Creates a temporary local replica that syncs with the remote libSQL database
   - Copies the synced replica to a timestamped backup file
   - Validates backup file exists and is non-empty before uploading

2. **Configuration** ([config.go](config.go))
   - Environment variable-based configuration with `.env` file support
   - Validates that DATABASE_PATH is a valid libSQL URL (libsql:// or https://*.turso.io)
   - Required fields: DATABASE_PATH, DATABASE_AUTH_TOKEN, AWS credentials, S3 bucket
   - Optional fields: cron schedule, backup prefix, bucket subfolder, object lock support

3. **URL Validation** ([download.go](download.go))
   - `ValidateDatabaseURL()` - Ensures database URL is a valid libSQL URL
   - `isLibSQLURL()` - Checks if URL matches libSQL patterns
   - **Rejects**: local file paths, HTTP URLs, S3 URLs, non-Turso HTTPS URLs

4. **S3 Upload** ([upload.go](upload.go))
   - Supports AWS S3 and S3-compatible storage (R2, B2, MinIO)
   - Optional MD5 checksum for object lock compliance
   - Configurable endpoint, region, and path-style addressing

5. **Scheduling** ([main.go](main.go))
   - Built-in cron scheduler (default: daily at 5 AM UTC)
   - Single-shot mode for external cron systems
   - Optional run-on-startup for immediate backups

### Key Functions

**CreateBackup (backup.go)**
```go
// Creates a backup using libSQL's embedded replica sync
// Returns: path to backup file, error
func CreateBackup(cfg *Config) (string, error)
```

**UploadToS3 (upload.go)**
```go
// Uploads backup file to S3-compatible storage
func UploadToS3(cfg *Config, backupFilePath string) error
```

**ValidateDatabaseURL (download.go)**
```go
// Validates that database URL is a libSQL URL
func ValidateDatabaseURL(dbURL string) error
```

## Configuration

### Environment Variables

**Required:**
- `DATABASE_PATH` - libSQL URL (libsql://db.turso.io or https://db.turso.io)
- `DATABASE_AUTH_TOKEN` - Turso authentication token
- `AWS_ACCESS_KEY_ID` - S3 access key
- `AWS_SECRET_ACCESS_KEY` - S3 secret key
- `AWS_S3_BUCKET` - S3 bucket name

**Optional:**
- `AWS_S3_REGION` - AWS region (default: us-east-1)
- `AWS_S3_ENDPOINT` - Custom S3 endpoint (for R2, MinIO, etc.)
- `AWS_S3_FORCE_PATH_STYLE` - Use path-style URLs (default: false)
- `BACKUP_CRON_SCHEDULE` - Cron pattern (default: "0 5 * * *")
- `BACKUP_FILE_PREFIX` - Backup filename prefix (default: "backup")
- `BUCKET_SUBFOLDER` - S3 subfolder path (default: "")
- `RUN_ON_STARTUP` - Run backup on startup (default: false)
- `SINGLE_SHOT_MODE` - Run once and exit (default: false)
- `SUPPORT_OBJECT_LOCK` - Enable MD5 checksums (default: false)

## Backup Process Flow

1. **Initialization**
   - Load and validate configuration
   - Ensure DATABASE_PATH is a valid libSQL URL

2. **Backup Creation**
   - Create temporary directory for embedded replica
   - Connect to libSQL database using URL and auth token
   - Sync embedded replica with remote database
   - Copy replica file to timestamped backup path
   - Validate backup file is non-empty

3. **S3 Upload**
   - Create AWS S3 client with configured credentials
   - Upload backup file to S3 bucket (with optional subfolder)
   - Calculate MD5 checksum if object lock is enabled
   - Report upload size and duration

4. **Cleanup**
   - Remove local backup file after successful upload
   - Remove temporary replica directory

## File Structure

```
.
├── backup.go          # libSQL backup logic using embedded replica
├── config.go          # Configuration loading and validation
├── download.go        # Database URL validation (libSQL only)
├── main.go            # Entry point, scheduler, CLI flags
├── upload.go          # S3 upload functionality
├── utils.go           # Helper functions (formatBytes, copyFile)
├── backup_test.go     # URL validation tests
├── config_test.go     # Configuration tests
├── download_test.go   # libSQL URL validation tests
├── utils_test.go      # Utility function tests
├── Dockerfile         # Multi-stage Docker build
├── go.mod             # Go module dependencies
└── README.md          # User documentation
```

## Dependencies

### Direct Dependencies
- `github.com/aws/aws-sdk-go-v2/*` - AWS S3 SDK
- `github.com/joho/godotenv` - .env file support
- `github.com/robfig/cron/v3` - Cron scheduler
- `github.com/tursodatabase/go-libsql` - libSQL client library

### Build Requirements
- Go 1.21+
- CGO enabled (required by libSQL)
- GCC/musl-dev (for static linking)

## Testing

### Test Coverage
- URL validation (libSQL vs non-libSQL URLs)
- Configuration loading and validation
- Boolean environment variable parsing
- Utility functions (formatBytes, etc.)

### Running Tests
```bash
go test -v ./...
```

### Key Test Cases
- Valid libSQL URLs (libsql://, https://*.turso.io)
- Invalid URLs (local paths, HTTP, S3, non-Turso HTTPS)
- Missing required configuration
- Default value handling
- Boolean parsing (true, false, 1, 0, empty)

## Building

### Local Build
```bash
CGO_ENABLED=1 go build -o libsql-s3-backup .
```

### Docker Build
```bash
docker build -t libsql-s3-backup .
```

The Dockerfile uses multi-stage builds:
1. **Build stage**: Alpine + Go + GCC for static binary compilation
2. **Runtime stage**: Minimal Alpine with ca-certificates

## Deployment Options

### Docker
```bash
docker run -d \
  -e DATABASE_PATH=libsql://mydb.turso.io \
  -e DATABASE_AUTH_TOKEN=token \
  -e AWS_ACCESS_KEY_ID=key \
  -e AWS_SECRET_ACCESS_KEY=secret \
  -e AWS_S3_BUCKET=bucket \
  libsql-s3-backup
```

### Railway
- One-click deploy via template button
- Configure environment variables in Railway dashboard
- Runs as a scheduled service

### Single-Shot Mode (External Cron)
```bash
export SINGLE_SHOT_MODE=true
./libsql-s3-backup
```

## Command-Line Flags

- `--version` - Print version and exit
- `--validate` - Validate configuration and exit (useful for debugging)

## Key Design Decisions

### libSQL-Only Approach
- **Why**: Simplifies codebase, clearer purpose, better for Turso users
- **Trade-off**: Can't backup local SQLite files directly
- **Benefit**: No need for file downloads, HTTP endpoints, or S3 source support

### Embedded Replica Strategy
- **Why**: libSQL's recommended backup method
- **How**: Creates local replica, syncs, copies file
- **Benefit**: Consistent, point-in-time snapshots

### URL Validation
- **Strict validation**: Only libsql:// and https://*.turso.io accepted
- **Fail fast**: Validation happens at config load, not at backup time
- **Clear errors**: Tells users exactly what URL format is required

### S3 Compatibility
- **Flexible**: Works with any S3-compatible storage
- **Configurable**: Custom endpoints, path-style URLs
- **Reliable**: MD5 checksums for object lock compliance

## Backup File Format

Files are named: `{prefix}-{timestamp}.db`

Example: `backup-2026-01-11T14-30-00-000Z.db`

- Prefix: Configurable via BACKUP_FILE_PREFIX
- Timestamp: UTC time in ISO 8601 format (safe for filesystems)
- Extension: Always `.db` (libSQL database file)

## Error Handling

### Configuration Errors
- Missing required environment variables → Fatal error with clear message
- Invalid libSQL URL → Descriptive validation error
- Invalid cron schedule → Error from cron library

### Runtime Errors
- Backup creation fails → Error logged, backup file preserved
- S3 upload fails → Error logged, local backup preserved for retry
- Sync failure → Error returned from libSQL library

### Cleanup Strategy
- Local backup deleted only after successful S3 upload
- Temporary replica directory cleaned up after backup creation
- Failed backups leave artifacts for debugging

## Monitoring and Logging

### Log Levels
- Info: Normal operation (startup, backup start/complete, upload progress)
- Warning: Non-fatal issues (cleanup failures)
- Fatal: Configuration errors, startup failures

### Key Metrics Logged
- Backup duration
- Backup file size
- S3 upload size
- Sync completion status

## Security Considerations

### Credentials
- Auth token required for libSQL access
- AWS credentials for S3 access
- No credentials stored in code or version control
- Use .env file (gitignored) or environment variables

### Network Security
- HTTPS for libSQL connections to Turso
- HTTPS for S3 uploads (configurable endpoint)
- CA certificates included in Docker image

### File Permissions
- Temporary files created in system temp directory
- Backup files cleaned up after upload
- No sensitive data in filenames

## Troubleshooting

### Common Issues

**"DATABASE_PATH validation failed"**
- Solution: Ensure URL is libsql:// or https://*.turso.io

**"failed to sync replica"**
- Check DATABASE_AUTH_TOKEN is correct
- Verify network connectivity to Turso
- Ensure database URL is accessible

**"S3 upload failed"**
- Verify AWS credentials are correct
- Check S3 bucket exists and is accessible
- For R2/MinIO: Set AWS_S3_ENDPOINT and AWS_S3_FORCE_PATH_STYLE

**CGO build errors**
- Ensure CGO_ENABLED=1
- Install gcc/musl-dev (Alpine) or build-essential (Ubuntu)

## Future Considerations

### Potential Enhancements
- Backup retention policies (automatic cleanup of old backups)
- Backup verification (download and validate backups)
- Multi-database support (backup multiple databases in one run)
- Webhooks/notifications (Slack, Discord, email on success/failure)
- Incremental backups (if libSQL adds support)
- Backup encryption (encrypt before S3 upload)
- Metrics export (Prometheus, CloudWatch)

### Current Limitations
- Only supports libSQL/Turso databases
- No built-in backup restoration (manual download from S3)
- No backup versioning beyond timestamps
- No compression (could reduce S3 costs)
- No progress reporting for large backups

## Version History

- **v1.0.0** - Initial libSQL-only release
  - Removed SQLite local file support
  - Removed HTTP/S3 download capabilities
  - Simplified to libSQL embedded replica only
  - Updated documentation and tests

## Contributing

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting
- Add tests for new functionality
- Update CLAUDE.md with architectural changes

### Testing Requirements
- All tests must pass (`go test -v ./...`)
- Add tests for new configuration options
- Add tests for new validation logic
- Maintain or improve test coverage

## Module Information

**Module**: `github.com/railwayapp-templates/libsql-s3-backups`
**Go Version**: 1.21+
**CGO Required**: Yes (for libSQL)
