package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tursodatabase/libsql-client-go/libsql"
)

// CreateBackup creates a backup based on the database type
func CreateBackup(cfg *Config) (string, error) {
	// Check if this is a libSQL database
	if isLibSQLURL(cfg.DatabasePath) {
		log.Println("Detected libSQL database URL")
		return createLibSQLBackup(cfg)
	}

	// For regular SQLite databases
	dbPath, isTemp, err := resolveDatabase(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to resolve database: %w", err)
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

	return createBackupFromPath(dbPath, cfg.BackupFilePrefix)
}

// createLibSQLBackup creates a backup of a libSQL database
func createLibSQLBackup(cfg *Config) (string, error) {
	// Generate timestamp-based filename
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05-000Z")
	filename := fmt.Sprintf("%s-%s.db", cfg.BackupFilePrefix, timestamp)
	backupPath := filepath.Join(os.TempDir(), filename)

	log.Printf("Connecting to libSQL database: %s", cfg.DatabasePath)

	// Create libSQL connector with auth token if provided
	var connector driver.Connector
	var err error

	if cfg.DatabaseAuthToken != "" {
		connector, err = libsql.NewConnector(cfg.DatabasePath, libsql.WithAuthToken(cfg.DatabaseAuthToken))
	} else {
		connector, err = libsql.NewConnector(cfg.DatabasePath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to create libSQL connector: %w", err)
	}

	// Open database connection
	db := sql.OpenDB(connector)
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return "", fmt.Errorf("failed to connect to libSQL database: %w", err)
	}

	log.Printf("Connected successfully, creating backup to %s", backupPath)

	// Execute VACUUM INTO command
	vacuumQuery := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	log.Printf("Executing: %s", vacuumQuery)

	if _, err := db.Exec(vacuumQuery); err != nil {
		return "", fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	// Validate backup file
	if err := validateBackupFile(backupPath); err != nil {
		return "", err
	}

	fileInfo, _ := os.Stat(backupPath)
	log.Printf("Backup created successfully: %s (size: %s)", backupPath, formatBytes(fileInfo.Size()))
	return backupPath, nil
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

	// Validate backup file
	if err := validateBackupFile(backupPath); err != nil {
		return "", err
	}

	fileInfo, _ := os.Stat(backupPath)
	log.Printf("Backup created successfully: %s (size: %s)", backupPath, formatBytes(fileInfo.Size()))
	return backupPath, nil
}

// validateBackupFile checks if the backup file is valid
func validateBackupFile(backupPath string) error {
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// Verify it's a valid SQLite database by attempting to open it
	db, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("backup file is not a valid SQLite database: %w", err)
	}
	defer db.Close()

	// Try to ping the database to ensure it's actually readable
	if err := db.Ping(); err != nil {
		return fmt.Errorf("backup file is corrupted or invalid: %w", err)
	}

	return nil
}
