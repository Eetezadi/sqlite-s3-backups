package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadToS3 uploads the backup file to S3
func UploadToS3(cfg *Config, backupFile string) error {
	ctx := context.Background()

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

	// Open backup file
	file, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Prepare S3 key
	filename := filepath.Base(backupFile)
	s3Key := filename
	if cfg.BucketSubfolder != "" {
		s3Key = strings.TrimSuffix(cfg.BucketSubfolder, "/") + "/" + filename
	}

	log.Printf("Uploading to s3://%s/%s", cfg.S3Bucket, s3Key)

	// Prepare upload input
	uploadInput := &s3.PutObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	}

	// Add ContentMD5 if object lock support is enabled
	if cfg.SupportObjectLock {
		log.Println("Computing MD5 hash for object lock support...")
		md5Hash, err := computeMD5(backupFile)
		if err != nil {
			return fmt.Errorf("failed to compute MD5: %w", err)
		}
		uploadInput.ContentMD5 = aws.String(md5Hash)
		log.Printf("MD5 hash computed: %s", md5Hash)
	}

	// Create uploader
	uploader := manager.NewUploader(s3Client)

	// Upload file
	result, err := uploader.Upload(ctx, uploadInput)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Upload successful: %s", result.Location)
	return nil
}

// createAWSConfig creates AWS SDK configuration
func createAWSConfig(ctx context.Context, cfg *Config) (aws.Config, error) {
	// Create credentials provider
	credsProvider := credentials.NewStaticCredentialsProvider(
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		"",
	)

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.AWSRegion),
		config.WithCredentialsProvider(credsProvider),
	)
	if err != nil {
		return aws.Config{}, err
	}

	return awsCfg, nil
}

// computeMD5 computes the MD5 hash of a file and returns it as base64
func computeMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	md5Sum := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(md5Sum), nil
}
