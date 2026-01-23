package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"vedio/shared/minio"

	miniosdk "github.com/minio/minio-go/v7"
)

// Service handles file storage operations.
// It currently implements ObjectStorage.
type Service struct {
	client        *minio.Client
	bucket        string
	presignClient *miniosdk.Client
	hostOverride  string
}

var _ ObjectStorage = (*Service)(nil)

// Option customizes the storage service behaviour.
type Option func(*Service)

// WithPresignClient sets a custom client for generating presigned URLs.
func WithPresignClient(client *miniosdk.Client) Option {
	return func(s *Service) {
		s.presignClient = client
	}
}

// WithHostOverride replaces the host in generated presigned URLs (e.g., for external access).
func WithHostOverride(host string) Option {
	return func(s *Service) {
		s.hostOverride = host
	}
}

// New creates a new storage service.
func New(client *minio.Client, opts ...Option) *Service {
	s := &Service{
		client:        client,
		bucket:        client.Bucket(),
		presignClient: client.PublicClient(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// PutObject uploads an object to MinIO.
func (s *Service) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		key,
		reader,
		size,
		miniosdk.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

// GetObject retrieves an object from MinIO.
func (s *Service) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(
		ctx,
		s.bucket,
		key,
		miniosdk.GetObjectOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	return obj, nil
}

// DeleteObject deletes an object from MinIO.
func (s *Service) DeleteObject(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(
		ctx,
		s.bucket,
		key,
		miniosdk.RemoveObjectOptions{},
	); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// PresignedGetURL generates a presigned URL for external access.
func (s *Service) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignedURL, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	if s.hostOverride != "" {
		parsedURL, err := url.Parse(presignedURL.String())
		if err != nil {
			return "", fmt.Errorf("failed to parse presigned URL: %w", err)
		}
		parsedURL.Host = s.hostOverride
		return parsedURL.String(), nil
	}

	return presignedURL.String(), nil
}

// ObjectExists checks whether an object exists without downloading it.
func (s *Service) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, miniosdk.StatObjectOptions{})
	if err == nil {
		return true, nil
	}

	responseErr := miniosdk.ToErrorResponse(err)
	if responseErr.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("failed to stat object: %w", err)
}
