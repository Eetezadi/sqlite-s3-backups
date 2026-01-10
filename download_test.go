package main

import (
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "HTTP URL",
			path:     "http://example.com/database.db",
			expected: true,
		},
		{
			name:     "HTTPS URL",
			path:     "https://example.com/database.db",
			expected: true,
		},
		{
			name:     "S3 URL",
			path:     "s3://bucket-name/path/to/database.db",
			expected: true,
		},
		{
			name:     "libSQL URL",
			path:     "libsql://mydb.turso.io",
			expected: true,
		},
		{
			name:     "local file path",
			path:     "/data/myapp.db",
			expected: false,
		},
		{
			name:     "relative path",
			path:     "./database.db",
			expected: false,
		},
		{
			name:     "Windows path",
			path:     "C:\\data\\database.db",
			expected: false,
		},
		{
			name:     "empty string",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isURL(tt.path)
			if result != tt.expected {
				t.Errorf("isURL(%q) = %v; expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsLibSQLURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "libSQL protocol",
			path:     "libsql://mydb.turso.io",
			expected: true,
		},
		{
			name:     "Turso HTTPS URL",
			path:     "https://mydb.turso.io",
			expected: true,
		},
		{
			name:     "Turso HTTPS with subdomain",
			path:     "https://my-db-name.turso.io",
			expected: true,
		},
		{
			name:     "regular HTTPS URL",
			path:     "https://example.com/database.db",
			expected: false,
		},
		{
			name:     "HTTP URL",
			path:     "http://example.com/database.db",
			expected: false,
		},
		{
			name:     "S3 URL",
			path:     "s3://bucket/database.db",
			expected: false,
		},
		{
			name:     "local path",
			path:     "/data/database.db",
			expected: false,
		},
		{
			name:     "empty string",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLibSQLURL(tt.path)
			if result != tt.expected {
				t.Errorf("isLibSQLURL(%q) = %v; expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestResolveDatabaseLogic(t *testing.T) {
	// Test that resolveDatabase correctly identifies different database sources
	// without actually downloading anything
	tests := []struct {
		name            string
		databasePath    string
		expectDownload  bool
		expectLibSQL    bool
	}{
		{
			name:           "local file path",
			databasePath:   "/data/myapp.db",
			expectDownload: false,
			expectLibSQL:   false,
		},
		{
			name:           "HTTP URL",
			databasePath:   "http://example.com/db.sqlite",
			expectDownload: true,
			expectLibSQL:   false,
		},
		{
			name:           "HTTPS URL",
			databasePath:   "https://example.com/database.db",
			expectDownload: true,
			expectLibSQL:   false,
		},
		{
			name:           "S3 URL",
			databasePath:   "s3://bucket/path/database.db",
			expectDownload: true,
			expectLibSQL:   false,
		},
		{
			name:           "libSQL URL",
			databasePath:   "libsql://mydb.turso.io",
			expectDownload: false,
			expectLibSQL:   true,
		},
		{
			name:           "Turso HTTPS URL",
			databasePath:   "https://mydb.turso.io",
			expectDownload: false,
			expectLibSQL:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isURLPath := isURL(tt.databasePath)
			isLibSQL := isLibSQLURL(tt.databasePath)
			needsDownload := isURLPath && !isLibSQL

			if needsDownload != tt.expectDownload {
				t.Errorf("Expected download=%v, got %v for path %s",
					tt.expectDownload, needsDownload, tt.databasePath)
			}

			if isLibSQL != tt.expectLibSQL {
				t.Errorf("Expected libSQL=%v, got %v for path %s",
					tt.expectLibSQL, isLibSQL, tt.databasePath)
			}
		})
	}
}
