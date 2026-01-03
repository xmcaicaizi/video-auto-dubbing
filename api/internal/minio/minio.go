package minio

import (
	"vedio/api/internal/config"
	sharedminio "vedio/shared/minio"
)

// Client is an alias to the shared MinIO client.
type Client = sharedminio.Client

// Option is an alias to the shared MinIO client options.
type Option = sharedminio.Option

// New creates a new MinIO client using the shared implementation.
func New(cfg config.MinIOConfig, opts ...sharedminio.Option) (*Client, error) {
	return sharedminio.New(cfg, opts...)
}
