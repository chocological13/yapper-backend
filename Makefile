.PHONY: run minio stop

# Run the Go application
run:
	go run ./cmd

# Run MinIO server
minio:
	minio server ~/minio-data --console-address ":9001"


