package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Database configuration
	DatabasePath      string
	DatabaseAuthToken string // For libSQL/Turso authentication

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

// LoadConfig loads and validates configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists (optional)
	_ = godotenv.Load()

	cfg := &Config{
		DatabasePath:       getEnv("DATABASE_PATH", ""),
		DatabaseAuthToken:  getEnv("DATABASE_AUTH_TOKEN", ""),
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
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if all required configuration is present
func (c *Config) Validate() error {
	if c.DatabasePath == "" {
		return fmt.Errorf("DATABASE_PATH is required")
	}
	if c.AWSAccessKeyID == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is required")
	}
	if c.AWSSecretAccessKey == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
	}
	if c.S3Bucket == "" {
		return fmt.Errorf("AWS_S3_BUCKET is required")
	}
	return nil
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
			return defaultValue
		}
		return boolVal
	}
	return defaultValue
}
