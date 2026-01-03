package minio

import (
	sharedminio "vedio/shared/minio"
	"vedio/worker/internal/config"
)

// Client is an alias to the shared MinIO client.
type Client = sharedminio.Client

// Option is an alias to the shared MinIO client options.
type Option = sharedminio.Option

// WithExistingBucketOnly re-exports the shared option to require an existing bucket.
var WithExistingBucketOnly = sharedminio.WithExistingBucketOnly

// New creates a new MinIO client using the shared implementation.
func New(cfg config.MinIOConfig, opts ...sharedminio.Option) (*Client, error) {
	return sharedminio.New(cfg, opts...)
}
