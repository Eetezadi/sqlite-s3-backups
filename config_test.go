package main

import (
	"os"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid local database path",
			config: Config{
				DatabasePath:       "/data/test.db",
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: false,
		},
		{
			name: "valid HTTP URL",
			config: Config{
				DatabasePath:       "https://example.com/database.db",
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: false,
		},
		{
			name: "valid S3 URL",
			config: Config{
				DatabasePath:       "s3://source-bucket/database.db",
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: false,
		},
		{
			name: "valid libSQL URL",
			config: Config{
				DatabasePath:       "libsql://mydb.turso.io",
				DatabaseAuthToken:  "test-token",
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: false,
		},
		{
			name: "missing database path",
			config: Config{
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: true,
			errorMsg:    "DATABASE_PATH is required",
		},
		{
			name: "missing AWS access key",
			config: Config{
				DatabasePath:       "/data/test.db",
				AWSSecretAccessKey: "test-secret",
				S3Bucket:           "test-bucket",
			},
			shouldError: true,
			errorMsg:    "AWS_ACCESS_KEY_ID is required",
		},
		{
			name: "missing AWS secret key",
			config: Config{
				DatabasePath:   "/data/test.db",
				AWSAccessKeyID: "test-key",
				S3Bucket:       "test-bucket",
			},
			shouldError: true,
			errorMsg:    "AWS_SECRET_ACCESS_KEY is required",
		},
		{
			name: "missing S3 bucket",
			config: Config{
				DatabasePath:       "/data/test.db",
				AWSAccessKeyID:     "test-key",
				AWSSecretAccessKey: "test-secret",
			},
			shouldError: true,
			errorMsg:    "AWS_S3_BUCKET is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear all environment variables
	os.Clearenv()

	// Set only required variables
	os.Setenv("DATABASE_PATH", "/data/test.db")
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("AWS_S3_BUCKET", "test-bucket")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test defaults
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("Expected default region 'us-east-1', got '%s'", cfg.AWSRegion)
	}

	if cfg.CronSchedule != "0 5 * * *" {
		t.Errorf("Expected default cron schedule '0 5 * * *', got '%s'", cfg.CronSchedule)
	}

	if cfg.BackupFilePrefix != "backup" {
		t.Errorf("Expected default prefix 'backup', got '%s'", cfg.BackupFilePrefix)
	}

	if cfg.RunOnStartup {
		t.Error("Expected RunOnStartup to be false by default")
	}

	if cfg.SingleShotMode {
		t.Error("Expected SingleShotMode to be false by default")
	}

	if cfg.SupportObjectLock {
		t.Error("Expected SupportObjectLock to be false by default")
	}

	if cfg.S3ForcePathStyle {
		t.Error("Expected S3ForcePathStyle to be false by default")
	}
}

func TestConfigBooleanParsing(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"1 as true", "1", true},
		{"false lowercase", "false", false},
		{"FALSE uppercase", "FALSE", false},
		{"0 as false", "0", false},
		{"empty string", "", false},
		{"random string", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("DATABASE_PATH", "/data/test.db")
			os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
			os.Setenv("AWS_S3_BUCKET", "test-bucket")
			os.Setenv("RUN_ON_STARTUP", tt.envValue)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.RunOnStartup != tt.expected {
				t.Errorf("For env value '%s', expected RunOnStartup=%v, got %v",
					tt.envValue, tt.expected, cfg.RunOnStartup)
			}
		})
	}
}

func TestConfigCustomValues(t *testing.T) {
	os.Clearenv()

	// Set custom values
	os.Setenv("DATABASE_PATH", "libsql://mydb.turso.io")
	os.Setenv("DATABASE_AUTH_TOKEN", "my-token")
	os.Setenv("AWS_ACCESS_KEY_ID", "custom-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "custom-secret")
	os.Setenv("AWS_S3_BUCKET", "custom-bucket")
	os.Setenv("AWS_S3_REGION", "eu-west-1")
	os.Setenv("AWS_S3_ENDPOINT", "https://s3.example.com")
	os.Setenv("BACKUP_CRON_SCHEDULE", "0 */6 * * *")
	os.Setenv("BACKUP_FILE_PREFIX", "mybackup")
	os.Setenv("BUCKET_SUBFOLDER", "backups/production")
	os.Setenv("RUN_ON_STARTUP", "true")
	os.Setenv("SINGLE_SHOT_MODE", "true")
	os.Setenv("SUPPORT_OBJECT_LOCK", "true")
	os.Setenv("AWS_S3_FORCE_PATH_STYLE", "true")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all custom values
	if cfg.DatabasePath != "libsql://mydb.turso.io" {
		t.Errorf("Expected DatabasePath 'libsql://mydb.turso.io', got '%s'", cfg.DatabasePath)
	}
	if cfg.DatabaseAuthToken != "my-token" {
		t.Errorf("Expected DatabaseAuthToken 'my-token', got '%s'", cfg.DatabaseAuthToken)
	}
	if cfg.AWSRegion != "eu-west-1" {
		t.Errorf("Expected AWSRegion 'eu-west-1', got '%s'", cfg.AWSRegion)
	}
	if cfg.S3Endpoint != "https://s3.example.com" {
		t.Errorf("Expected S3Endpoint 'https://s3.example.com', got '%s'", cfg.S3Endpoint)
	}
	if cfg.CronSchedule != "0 */6 * * *" {
		t.Errorf("Expected CronSchedule '0 */6 * * *', got '%s'", cfg.CronSchedule)
	}
	if cfg.BackupFilePrefix != "mybackup" {
		t.Errorf("Expected BackupFilePrefix 'mybackup', got '%s'", cfg.BackupFilePrefix)
	}
	if cfg.BucketSubfolder != "backups/production" {
		t.Errorf("Expected BucketSubfolder 'backups/production', got '%s'", cfg.BucketSubfolder)
	}
	if !cfg.RunOnStartup {
		t.Error("Expected RunOnStartup to be true")
	}
	if !cfg.SingleShotMode {
		t.Error("Expected SingleShotMode to be true")
	}
	if !cfg.SupportObjectLock {
		t.Error("Expected SupportObjectLock to be true")
	}
	if !cfg.S3ForcePathStyle {
		t.Error("Expected S3ForcePathStyle to be true")
	}
}
