package ports

import "context"

// StorageUploader stages raw uploads and deletes ephemeral objects (S3-compatible).
type StorageUploader interface {
	StageRaw(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
}
