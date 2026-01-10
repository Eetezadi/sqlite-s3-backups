package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// resolveDatabase returns the local path to the database, downloading if necessary
// Returns: (path, isTemporary, error)
func resolveDatabase(cfg *Config) (string, bool, error) {
	dbSource := cfg.DatabasePath

	// Check if it's a URL (but not libSQL, which is handled separately)
	if isURL(dbSource) && !isLibSQLURL(dbSource) {
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
			return "", false, fmt.Errorf("unsupported URL scheme: %s (supported: http, https, s3, libsql)", parsedURL.Scheme)
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

// isURL checks if a string is a URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "s3://") ||
		strings.HasPrefix(s, "libsql://")
}

// isLibSQLURL checks if a string is a libSQL URL
func isLibSQLURL(s string) bool {
	return strings.HasPrefix(s, "libsql://") ||
		strings.HasPrefix(s, "https://") && strings.Contains(s, ".turso.io")
}
