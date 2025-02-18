package mediaservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"mime/multipart"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Service interface {
	UploadMedia(ctx context.Context, file *multipart.FileHeader) (string, error)
	DeleteMedia(ctx context.Context, url string) error
}

type mediaService struct {
	s3Client   *s3.Client
	bucketName string
	baseURL    string
}

func NewMediaService(cfg Config) (Service, error) {
	// Create AWS config
	awsCfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		),
		Region: cfg.Region,
		//// For MinIO/local development, we need to disable HTTPS
		//RetryMaxAttempts: 3,
		//HTTPClient:       nil, // Use default
	}

	// Initialize S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.UseSSL {
			o.BaseEndpoint = nil // Remove manual endpoint for AWS S3 / Cloudflare R2
		} else {
			o.BaseEndpoint = aws.String(cfg.Endpoint) // For MinIO (HTTP)
			o.UsePathStyle = true                     // Required for MinIO
		}
	})

	return &mediaService{
		s3Client:   client,
		bucketName: cfg.BucketName,
		baseURL:    cfg.Endpoint + "/" + cfg.BucketName,
	}, nil
}

func generateShortID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (s *mediaService) UploadMedia(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s%s", generateShortID(), ext)

	// Open file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Upload to S3-compatible storage
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        src,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return fmt.Sprintf("%s%s", s.baseURL, file.Filename), nil
}

func (s *mediaService) DeleteMedia(ctx context.Context, url string) error {
	// Extract the key from the URL
	// If URL is like "http://localhost:9000/yap-media/filename.jpg"
	// we need to extract "filename.jpg"
	key := url[len(s.baseURL)+1:]

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}

	return nil
}
