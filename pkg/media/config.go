package media

import (
	"os"
)

var (
	MaxFileSize int64 = 10 * 1024 * 1024
	ValidTypes        = []string{"image/jpeg", "image/png", "image/gif", "video/mp4", "video/quicktime"}
)

type Config struct {
	BucketName      string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool // to differentiate local and prod
}

func LoadConfigFromEnv() Config {
	return Config{
		BucketName:      getEnvOrDefault("MEDIA_BUCKET_NAME", "yap-media"),
		Region:          getEnvOrDefault("MEDIA_REGION", "us-east-1"),
		Endpoint:        getEnvOrDefault("MEDIA_ENDPOINT", "http://localhost:9000"),
		AccessKeyID:     getEnvOrDefault("MEDIA_ACCESS_KEY_ID", "minioadmin"),
		SecretAccessKey: getEnvOrDefault("MEDIA_SECRET_ACCESS_KEY", "minioadmin"),
		UseSSL:          getEnvOrDefault("MEDIA_USE_SSL", "false") == "true",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
