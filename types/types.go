package types

import (
	"context"
	"io"
	"time"
)

// FileStorage defines the core interface for file storage operations
type FileStorage interface {
	UploadFile(ctx context.Context, req *UploadRequest) (*UploadResponse, error)
	DeleteFile(ctx context.Context, path string) error
	GetFileURL(path string) string
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
}

// SnapshotStorage defines the interface for snapshot-specific operations
type SnapshotStorage interface {
	UploadSnapshot(ctx context.Context, file io.Reader, filename string) (string, error)
	DeleteSnapshot(ctx context.Context, path string) error
	GetSnapshotURL(path string) string
}

// AuditableStorage extends FileStorage with audit capabilities
type AuditableStorage interface {
	FileStorage
	UploadFileWithAudit(ctx context.Context, req *UploadRequest, userID string) (*UploadResponse, error)
	DeleteFileWithAudit(ctx context.Context, path, userID string) error
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	File        io.Reader              `json:"-"`
	Filename    string                 `json:"filename"`
	ContentType string                 `json:"content_type"`
	FileSize    int64                  `json:"file_size"`
	BucketPath  string                 `json:"bucket_path"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UploadResponse represents the response from a file upload
type UploadResponse struct {
	Path         string                 `json:"path"`
	URL          string                 `json:"url"`
	Size         int64                  `json:"size"`
	ContentType  string                 `json:"content_type"`
	ETag         string                 `json:"etag"`
	LastModified time.Time              `json:"last_modified"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FileInfo represents information about a stored file
type FileInfo struct {
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	ContentType  string                 `json:"content_type"`
	ETag         string                 `json:"etag"`
	LastModified time.Time              `json:"last_modified"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FileValidationResult represents the result of file validation
type FileValidationResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// RetryConfig represents configuration for retry operations
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     float64       `json:"backoff"`
}

// UploadProgress represents upload progress information
type UploadProgress struct {
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`
	Percentage       float64 `json:"percentage"`
	Speed            int64   `json:"speed_bytes_per_second"`
	ETA              int64   `json:"eta_seconds"`
}

// StorageStats represents storage statistics
type StorageStats struct {
	TotalFiles     int64     `json:"total_files"`
	TotalSize      int64     `json:"total_size"`
	UsedSpace      int64     `json:"used_space"`
	AvailableSpace int64     `json:"available_space"`
	LastUpdated    time.Time `json:"last_updated"`
}
