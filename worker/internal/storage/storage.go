package storage

import (
	sharedstorage "vedio/shared/storage"
	"vedio/worker/internal/minio"
)

// Service is an alias to the shared storage service.
type Service = sharedstorage.Service

// Option re-exports the shared storage options.
type Option = sharedstorage.Option

// WithHostOverride re-exports the shared host override option.
var WithHostOverride = sharedstorage.WithHostOverride

// New creates a new storage service using the shared implementation.
func New(client *minio.Client, opts ...sharedstorage.Option) *Service {
	return sharedstorage.New(client, opts...)
}
