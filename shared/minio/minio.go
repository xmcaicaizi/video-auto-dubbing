package minio

import (
	"context"
	"fmt"

	"vedio/shared/config"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps the MinIO client and exposes the configured bucket.
type Client struct {
	*miniosdk.Client
	publicClient *miniosdk.Client
	bucket       string
}

// Option customizes client initialization.
type Option func(*options)

type options struct {
	requireExistingBucket bool
}

// WithExistingBucketOnly requires the bucket to exist instead of creating it.
func WithExistingBucketOnly() Option {
	return func(o *options) {
		o.requireExistingBucket = true
	}
}

// New creates a new MinIO client with optional bucket validation behaviour.
func New(cfg config.MinIOConfig, opts ...Option) (*Client, error) {
	settings := options{}
	for _, opt := range opts {
		opt(&settings)
	}

	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	var publicClient *miniosdk.Client
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		publicClient, err = miniosdk.New(cfg.PublicEndpoint, &miniosdk.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create public MinIO client: %w", err)
		}
	} else {
		publicClient = client
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if settings.requireExistingBucket {
			return nil, fmt.Errorf("bucket %s does not exist", cfg.Bucket)
		}

		if err := client.MakeBucket(ctx, cfg.Bucket, miniosdk.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &Client{
		Client:       client,
		publicClient: publicClient,
		bucket:       cfg.Bucket,
	}, nil
}

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string {
	return c.bucket
}

// PublicClient returns the client used for presigned URL generation.
func (c *Client) PublicClient() *miniosdk.Client {
	return c.publicClient
}
