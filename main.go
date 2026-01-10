package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
)

// Config holds all application configuration
type Config struct {
	// SQLite configuration
	DatabasePath string

	// AWS S3 configuration
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	S3Bucket           string
	S3Endpoint         string
	S3ForcePathStyle   bool
	BucketSubfolder    string

	// Backup configuration
	BackupFilePrefix  string
	SupportObjectLock bool

	// Scheduling configuration
	CronSchedule   string
	RunOnStartup   bool
	SingleShotMode bool
}

func main() {
	// Load .env file if it exists (optional)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Println("SQLite S3 Backup Service starting...")
	log.Printf("Database: %s", cfg.DatabasePath)
	log.Printf("S3 Bucket: %s/%s", cfg.S3Bucket, cfg.BucketSubfolder)
	log.Printf("Schedule: %s", cfg.CronSchedule)
	log.Printf("Single-shot mode: %v", cfg.SingleShotMode)

	// Run backup on startup if configured
	if cfg.RunOnStartup {
		log.Println("Running backup on startup...")
		if err := runBackup(cfg); err != nil {
			log.Fatalf("Startup backup failed: %v", err)
		}
	}

	// If single-shot mode, exit after startup backup
	if cfg.SingleShotMode {
		if !cfg.RunOnStartup {
			log.Println("Single-shot mode: Running backup once...")
			if err := runBackup(cfg); err != nil {
				log.Fatalf("Backup failed: %v", err)
			}
		}
		log.Println("Single-shot mode: Backup complete, exiting")
		return
	}

	// Set up cron scheduler
	c := cron.New()
	_, err = c.AddFunc(cfg.CronSchedule, func() {
		log.Println("Cron job triggered, starting backup...")
		if err := runBackup(cfg); err != nil {
			log.Printf("Backup failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to schedule backup: %v", err)
	}

	c.Start()
	log.Printf("Backup scheduled with cron pattern: %s", cfg.CronSchedule)
	log.Println("Service is running. Press Ctrl+C to exit.")

	// Keep the program running
	select {}
}

// loadConfig loads and validates configuration from environment variables
func loadConfig() (*Config, error) {
	cfg := &Config{
		DatabasePath:       getEnv("DATABASE_PATH", ""),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSRegion:          getEnv("AWS_S3_REGION", "us-east-1"),
		S3Bucket:           getEnv("AWS_S3_BUCKET", ""),
		S3Endpoint:         getEnv("AWS_S3_ENDPOINT", ""),
		S3ForcePathStyle:   getBoolEnv("AWS_S3_FORCE_PATH_STYLE", false),
		BucketSubfolder:    getEnv("BUCKET_SUBFOLDER", ""),
		BackupFilePrefix:   getEnv("BACKUP_FILE_PREFIX", "backup"),
		SupportObjectLock:  getBoolEnv("SUPPORT_OBJECT_LOCK", false),
		CronSchedule:       getEnv("BACKUP_CRON_SCHEDULE", "0 5 * * *"),
		RunOnStartup:       getBoolEnv("RUN_ON_STARTUP", false),
		SingleShotMode:     getBoolEnv("SINGLE_SHOT_MODE", false),
	}

	// Validate required fields
	if cfg.DatabasePath == "" {
		return nil, fmt.Errorf("DATABASE_PATH is required")
	}
	if cfg.AWSAccessKeyID == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID is required")
	}
	if cfg.AWSSecretAccessKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
	}
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Warning: Invalid boolean value for %s: %s, using default: %v", key, value, defaultValue)
			return defaultValue
		}
		return boolVal
	}
	return defaultValue
}

// runBackup performs the complete backup and upload process
func runBackup(cfg *Config) error {
	startTime := time.Now()
	log.Println("Starting backup process...")

	// Check if database path is a URL and download if needed
	dbPath, isTemp, err := resolveDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve database: %w", err)
	}

	// Clean up downloaded database file if it was temporary
	if isTemp {
		defer func() {
			if err := os.Remove(dbPath); err != nil {
				log.Printf("Warning: Failed to remove temporary database file %s: %v", dbPath, err)
			} else {
				log.Printf("Cleaned up temporary database file: %s", dbPath)
			}
		}()
	}

	// Create backup file
	backupFile, err := createBackupFromPath(dbPath, cfg.BackupFilePrefix)
	if err != nil {
		return fmt.Errorf("backup creation failed: %w", err)
	}

	// Upload to S3
	if err := uploadToS3(cfg, backupFile); err != nil {
		return fmt.Errorf("S3 upload failed (backup file preserved at %s): %w", backupFile, err)
	}

	// Clean up backup file only after successful upload
	if err := os.Remove(backupFile); err != nil {
		log.Printf("Warning: Failed to remove backup file %s: %v", backupFile, err)
	} else {
		log.Printf("Cleaned up local backup file: %s", backupFile)
	}

	duration := time.Since(startTime)
	log.Printf("Backup completed successfully in %v", duration)
	return nil
}

// resolveDatabase returns the local path to the database, downloading if necessary
// Returns: (path, isTemporary, error)
func resolveDatabase(cfg *Config) (string, bool, error) {
	dbSource := cfg.DatabasePath

	// Check if it's a URL
	if isURL(dbSource) {
		log.Printf("Database source is a URL: %s", dbSource)
		parsedURL, err := url.Parse(dbSource)
		if err != nil {
			return "", false, fmt.Errorf("invalid URL: %w", err)
		}

		// Create temp file for download
		tempFile, err := os.CreateTemp(os.TempDir(), "source-db-*.db")
		if err != nil {
			return "", false, fmt.Errorf("failed to create temp file: %w", err)
		}
		tempPath := tempFile.Name()
		tempFile.Close()

		// Download based on scheme
		switch parsedURL.Scheme {
		case "http", "https":
			log.Printf("Downloading database from HTTP(S)...")
			if err := downloadHTTP(dbSource, tempPath); err != nil {
				os.Remove(tempPath)
				return "", false, fmt.Errorf("HTTP download failed: %w", err)
			}
		case "s3":
			log.Printf("Downloading database from S3...")
			if err := downloadS3(cfg, parsedURL, tempPath); err != nil {
				os.Remove(tempPath)
				return "", false, fmt.Errorf("S3 download failed: %w", err)
			}
		default:
			os.Remove(tempPath)
			return "", false, fmt.Errorf("unsupported URL scheme: %s (supported: http, https, s3)", parsedURL.Scheme)
		}

		log.Printf("Database downloaded successfully to: %s", tempPath)
		return tempPath, true, nil
	}

	// It's a local file path
	if _, err := os.Stat(dbSource); err != nil {
		return "", false, fmt.Errorf("database file not found: %w", err)
	}

	return dbSource, false, nil
}

// isURL checks if a string is a URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "s3://")
}

// downloadHTTP downloads a file from an HTTP/HTTPS URL
func downloadHTTP(urlStr, destPath string) error {
	resp, err := http.Get(urlStr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Downloaded %s from HTTP", formatBytes(written))
	return nil
}

// downloadS3 downloads a file from S3
func downloadS3(cfg *Config, parsedURL *url.URL, destPath string) error {
	ctx := context.Background()

	// Parse S3 URL: s3://bucket/key
	bucket := parsedURL.Host
	key := strings.TrimPrefix(parsedURL.Path, "/")

	if bucket == "" || key == "" {
		return fmt.Errorf("invalid S3 URL format, expected s3://bucket/key")
	}

	// Create AWS config
	awsCfg, err := createAWSConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = cfg.S3ForcePathStyle
	})

	// Create downloader
	downloader := manager.NewDownloader(s3Client)

	// Create output file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download
	log.Printf("Downloading s3://%s/%s", bucket, key)
	numBytes, err := downloader.Download(ctx, out, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	log.Printf("Downloaded %s from S3", formatBytes(numBytes))
	return nil
}

// createBackupFromPath creates a SQLite backup using VACUUM INTO
func createBackupFromPath(dbPath, backupPrefix string) (string, error) {
	// Generate timestamp-based filename
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05-000Z")
	filename := fmt.Sprintf("%s-%s.db", backupPrefix, timestamp)
	backupPath := filepath.Join(os.TempDir(), filename)

	log.Printf("Creating backup of %s to %s", dbPath, backupPath)

	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}

	// Execute VACUUM INTO command
	vacuumQuery := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	log.Printf("Executing: %s", vacuumQuery)

	if _, err := db.Exec(vacuumQuery); err != nil {
		return "", fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Validate backup file
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("backup file is empty")
	}

	log.Printf("Backup created successfully: %s (size: %s)", backupPath, formatBytes(fileInfo.Size()))
	return backupPath, nil
}

// uploadToS3 uploads the backup file to S3
func uploadToS3(cfg *Config, backupFile string) error {
	ctx := context.Background()

	// Create AWS config
	awsCfg, err := createAWSConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		}
		o.UsePathStyle = cfg.S3ForcePathStyle
	})

	// Open backup file
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Prepare S3 key
	filename := filepath.Base(backupFile)
	s3Key := filename
	if cfg.BucketSubfolder != "" {
		s3Key = strings.TrimSuffix(cfg.BucketSubfolder, "/") + "/" + filename
	}

	log.Printf("Uploading to s3://%s/%s", cfg.S3Bucket, s3Key)

	// Prepare upload input
	uploadInput := &s3.PutObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	}

	// Add ContentMD5 if object lock support is enabled
	if cfg.SupportObjectLock {
		log.Println("Computing MD5 hash for object lock support...")
		md5Hash, err := computeMD5(backupFile)
		if err != nil {
			return fmt.Errorf("failed to compute MD5: %w", err)
		}
		uploadInput.ContentMD5 = aws.String(md5Hash)
		log.Printf("MD5 hash computed: %s", md5Hash)
	}

	// Create uploader
	uploader := manager.NewUploader(s3Client)

	// Upload file
	result, err := uploader.Upload(ctx, uploadInput)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Upload successful: %s", result.Location)
	return nil
}

// createAWSConfig creates AWS SDK configuration
func createAWSConfig(ctx context.Context, cfg *Config) (aws.Config, error) {
	// Create credentials provider
	credsProvider := credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	)

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.AWSRegion),
		config.WithCredentialsProvider(credsProvider),
	)
	if err != nil {
		return aws.Config{}, err
	}

	return awsCfg, nil
}

// computeMD5 computes the MD5 hash of a file and returns it as base64
func computeMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	md5Sum := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(md5Sum), nil
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
