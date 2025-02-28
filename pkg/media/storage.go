package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"mime/multipart"
	"time"
)

type StorageService interface {
	Upload(ctx context.Context, file *multipart.FileHeader, folderName string) (string, error)
	Delete(ctx context.Context, url string) error
}

type s3StorageService struct {
	s3Client   *s3.Client
	bucketName string
	baseURL    string
}

func NewStorageService(cfg Config) (StorageService, error) {
	return &s3StorageService{
		s3Client:   createS3Client(cfg),
		bucketName: cfg.BucketName,
		baseURL:    cfg.Endpoint + "/" + cfg.BucketName,
	}, nil
}

func (s *s3StorageService) Upload(ctx context.Context, file *multipart.FileHeader, folderName string) (string, error) {
	// Generate unique filename
	filename := generateFilename(folderName)

	// Open file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Upload to S3-compatible storage
	contentType := getContentType(file)

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        src,
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.baseURL, filename), nil
}

func (s *s3StorageService) Delete(ctx context.Context, url string) error {
	// Extract the key from the URL
	// URL is like "http://localhost:9000/yap-media/filename.jpg"
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

// -------------------- 🔽 HELPER FUNCTIONS BELOW 🔽 --------------------

func createS3Client(cfg Config) *s3.Client {
	awsCfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		),
		Region: cfg.Region,
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if !cfg.UseSSL {
			// config for MinIO
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
}

func generateShortID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func generateFilename(folderName string) string {
	return fmt.Sprintf("%s/%s-%d", folderName, generateShortID(), time.Now().Unix())
}

func getContentType(file *multipart.FileHeader) string {
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}
