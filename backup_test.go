package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestValidateBackupFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test valid SQLite database
	t.Run("valid SQLite database", func(t *testing.T) {
		validDB := filepath.Join(tempDir, "valid.db")
		db, err := sql.Open("sqlite3", validDB)
		if err != nil {
			t.Fatalf("Failed to create test database: %v", err)
		}
		_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
		if err != nil {
			t.Fatalf("Failed to create test table: %v", err)
		}
		db.Close()

		if err := validateBackupFile(validDB); err != nil {
			t.Errorf("Expected valid database to pass validation, got error: %v", err)
		}
	})

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistent := filepath.Join(tempDir, "nonexistent.db")
		err := validateBackupFile(nonExistent)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	// Test empty file
	t.Run("empty file", func(t *testing.T) {
		emptyFile := filepath.Join(tempDir, "empty.db")
		if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		err := validateBackupFile(emptyFile)
		if err == nil {
			t.Error("Expected error for empty file, got nil")
		}
		if !strings.Contains(err.Error(), "is empty") {
			t.Errorf("Expected 'is empty' error, got: %v", err)
		}
	})

	// Test invalid SQLite file
	t.Run("invalid SQLite file", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid.db")
		if err := os.WriteFile(invalidFile, []byte("not a sqlite database"), 0644); err != nil {
			t.Fatalf("Failed to create invalid file: %v", err)
		}

		err := validateBackupFile(invalidFile)
		if err == nil {
			t.Error("Expected error for invalid SQLite file, got nil")
		}
	})
}

func TestBackupFilenameFormat(t *testing.T) {
	// Test that backup filenames follow expected format
	// by creating an actual backup and checking its filename
	tempDir, err := os.MkdirTemp("", "backup-name-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test source database
	sourceDB := filepath.Join(tempDir, "source.db")
	db, err := sql.Open("sqlite3", sourceDB)
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}
	db.Exec("CREATE TABLE test (id INTEGER)")
	db.Close()

	// Create backup with default prefix
	backupFile, err := createBackupFromPath(sourceDB, "backup")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer os.Remove(backupFile)

	filename := filepath.Base(backupFile)

	// Check prefix
	if !strings.HasPrefix(filename, "backup-") {
		t.Errorf("Expected filename to start with 'backup-', got: %s", filename)
	}

	// Check extension
	if !strings.HasSuffix(filename, ".db") {
		t.Errorf("Expected filename to end with '.db', got: %s", filename)
	}
}

func TestBackupFilenameWithCustomPrefix(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup-prefix-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDB := filepath.Join(tempDir, "source.db")
	db, err := sql.Open("sqlite3", sourceDB)
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}
	db.Exec("CREATE TABLE test (id INTEGER)")
	db.Close()

	backupFile, err := createBackupFromPath(sourceDB, "myapp")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer os.Remove(backupFile)

	filename := filepath.Base(backupFile)
	if !strings.HasPrefix(filename, "myapp-") {
		t.Errorf("Expected filename to start with 'myapp-', got: %s", filename)
	}
}

func TestCreateBackupFromPath(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "backup-create-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test source database
	sourceDB := filepath.Join(tempDir, "source.db")
	db, err := sql.Open("sqlite3", sourceDB)
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}

	// Create test data
	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);
		INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com');
		INSERT INTO users (name, email) VALUES ('Bob', 'bob@example.com');
	`)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}
	db.Close()

	// Create backup
	backupFile, err := createBackupFromPath(sourceDB, "test")
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer os.Remove(backupFile)

	// Verify backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// Verify backup is a valid SQLite database
	if err := validateBackupFile(backupFile); err != nil {
		t.Errorf("Backup file validation failed: %v", err)
	}

	// Verify backup contains the data
	backupDB, err := sql.Open("sqlite3", backupFile)
	if err != nil {
		t.Fatalf("Failed to open backup database: %v", err)
	}
	defer backupDB.Close()

	var count int
	err = backupDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query backup database: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 rows in backup, got %d", count)
	}
}

func TestCreateBackupFromPathNonExistent(t *testing.T) {
	_, err := createBackupFromPath("/nonexistent/database.db", "test")
	if err == nil {
		t.Error("Expected error for non-existent source database, got nil")
	}
}
