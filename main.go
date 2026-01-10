package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

const version = "1.0.0"

func main() {
	// Command line flags
	versionFlag := flag.Bool("version", false, "Print version and exit")
	validateFlag := flag.Bool("validate", false, "Validate configuration and exit")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("sqlite-s3-backup version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Handle validate flag
	if *validateFlag {
		log.Println("Configuration validation successful!")
		log.Printf("Database: %s", cfg.DatabasePath)
		log.Printf("S3 Bucket: %s/%s", cfg.S3Bucket, cfg.BucketSubfolder)
		log.Printf("Schedule: %s", cfg.CronSchedule)
		log.Printf("Single-shot mode: %v", cfg.SingleShotMode)
		os.Exit(0)
	}

	log.Printf("SQLite S3 Backup Service v%s starting...", version)
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

// runBackup performs the complete backup and upload process
func runBackup(cfg *Config) error {
	startTime := time.Now()
	log.Println("Starting backup process...")

	// Create backup
	backupFile, err := CreateBackup(cfg)
	if err != nil {
		return err
	}

	// Upload to S3
	if err := UploadToS3(cfg, backupFile); err != nil {
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
