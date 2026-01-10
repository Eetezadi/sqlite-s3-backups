package main

import (
	"fmt"
	"strings"
)

// ValidateDatabaseURL checks if the database URL is a valid libSQL URL
func ValidateDatabaseURL(dbURL string) error {
	if !isLibSQLURL(dbURL) {
		return fmt.Errorf("invalid database URL: must be a libSQL URL (libsql:// or https://*.turso.io)")
	}
	return nil
}

// isLibSQLURL checks if a string is a libSQL URL
func isLibSQLURL(s string) bool {
	return strings.HasPrefix(s, "libsql://") ||
		(strings.HasPrefix(s, "https://") && strings.Contains(s, ".turso.io"))
}
