package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"
)

// CreateBackup creates a backup of a libSQL database using embedded replica
func CreateBackup(cfg *Config) (string, error) {
	// Generate timestamp-based filename
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05-000Z")
	filename := fmt.Sprintf("%s-%s.db", cfg.BackupFilePrefix, timestamp)
	backupPath := filepath.Join(os.TempDir(), filename)

	log.Printf("Creating embedded replica for libSQL database: %s", cfg.DatabasePath)

	// Create temporary directory for the replica
	tempDir, err := os.MkdirTemp("", "libsql-replica-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp dir after we're done

	replicaPath := filepath.Join(tempDir, "replica.db")

	// Create embedded replica connector
	var connector *libsql.Connector
	if cfg.DatabaseAuthToken != "" {
		connector, err = libsql.NewEmbeddedReplicaConnector(
			replicaPath,
			cfg.DatabasePath,
			libsql.WithAuthToken(cfg.DatabaseAuthToken),
		)
	} else {
		connector, err = libsql.NewEmbeddedReplicaConnector(
			replicaPath,
			cfg.DatabasePath,
		)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create embedded replica connector: %w", err)
	}
	defer connector.Close()

	// Sync the remote database to local replica
	log.Println("Syncing remote database to local replica...")
	if _, err := connector.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync database: %w", err)
	}

	log.Println("Sync completed, copying replica to backup location...")

	// Copy the replica file to the backup path
	if err := copyFile(replicaPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to copy replica to backup: %w", err)
	}

	// Validate backup file exists and is not empty
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat backup file: %w", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("backup file is empty")
	}
	log.Printf("Backup created successfully: %s (size: %s)", backupPath, formatBytes(fileInfo.Size()))
	return backupPath, nil
}
