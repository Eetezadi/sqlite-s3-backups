package main

import (
	"testing"
)

func TestValidateDatabaseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid libsql URL",
			url:     "libsql://my-database.turso.io",
			wantErr: false,
		},
		{
			name:    "valid Turso HTTPS URL",
			url:     "https://my-database.turso.io",
			wantErr: false,
		},
		{
			name:    "invalid HTTP URL",
			url:     "http://example.com/db.db",
			wantErr: true,
		},
		{
			name:    "invalid S3 URL",
			url:     "s3://bucket/database.db",
			wantErr: true,
		},
		{
			name:    "invalid local path",
			url:     "/path/to/database.db",
			wantErr: true,
		},
		{
			name:    "invalid HTTPS URL (non-Turso)",
			url:     "https://example.com/database.db",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabaseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDatabaseURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsLibSQLURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "libsql:// protocol",
			url:  "libsql://my-database.turso.io",
			want: true,
		},
		{
			name: "https:// with turso.io",
			url:  "https://my-database.turso.io",
			want: true,
		},
		{
			name: "https:// without turso.io",
			url:  "https://example.com",
			want: false,
		},
		{
			name: "http:// with turso.io",
			url:  "http://my-database.turso.io",
			want: false,
		},
		{
			name: "local file path",
			url:  "/path/to/database.db",
			want: false,
		},
		{
			name: "s3:// URL",
			url:  "s3://bucket/database.db",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLibSQLURL(tt.url); got != tt.want {
				t.Errorf("isLibSQLURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
