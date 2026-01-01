package minio

import (
	"context"
	"fmt"

	"vedio/api/internal/config"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps the MinIO client.
type Client struct {
	*miniosdk.Client
	publicClient *miniosdk.Client // 用于生成浏览器可达的预签名 URL
	bucket       string
}

// New creates a new MinIO client.
func New(cfg config.MinIOConfig) (*Client, error) {
	// 创建内部客户端（用于容器间通信）
	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// 创建公共客户端（用于生成浏览器可达的预签名 URL）
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
		// 如果未指定公共端点或与内部端点相同，使用内部客户端
		publicClient = client
	}

	// Check if bucket exists, create if not
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
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

// Bucket returns the bucket name.
func (c *Client) Bucket() string {
	return c.bucket
}

// PublicClient returns the public MinIO client for generating presigned URLs.
func (c *Client) PublicClient() *miniosdk.Client {
	return c.publicClient
}

