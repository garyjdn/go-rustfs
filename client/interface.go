package client

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/garyjdn/go-rustfs/types"
)

// FileStorage defines the core interface for file storage operations
type FileStorage interface {
	UploadFile(ctx context.Context, req *types.UploadRequest) (*types.UploadResponse, error)
	DeleteFile(ctx context.Context, path string) error
	GetFileURL(path string) string
	GetFileInfo(ctx context.Context, path string) (*types.FileInfo, error)
}

// SnapshotStorage defines the interface for snapshot-specific operations
type SnapshotStorage interface {
	UploadSnapshot(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error)
	DeleteSnapshot(ctx context.Context, path string) error
	GetSnapshotURL(path string) string
}

// AuditableStorage extends FileStorage with audit capabilities
type AuditableStorage interface {
	FileStorage
	UploadFileWithAudit(ctx context.Context, req *types.UploadRequest, userID string) (*types.UploadResponse, error)
	DeleteFileWithAudit(ctx context.Context, path, userID string) error
}

// ProgressCallback defines a callback for upload progress
type ProgressCallback func(progress *types.UploadProgress)

// UploadOptions defines options for file upload
type UploadOptions struct {
	ProgressCallback  ProgressCallback
	EnableCompression bool
	EnableEncryption  bool
	Metadata          map[string]interface{}
}

// ClientOptions defines options for client initialization
type ClientOptions struct {
	Timeout        *types.RetryConfig
	EnableCache    bool
	CacheSize      int
	EnableMetrics  bool
	UserAgent      string
	MaxConcurrency int
}

// StorageStats defines storage statistics interface
type StorageStats interface {
	GetStorageStats(ctx context.Context) (*types.StorageStats, error)
	GetUsageByUser(ctx context.Context, userID string) (int64, error)
	GetUsageByType(ctx context.Context, contentType string) (int64, error)
}

// HealthChecker defines health check interface
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
	GetVersion(ctx context.Context) (string, error)
	GetCapabilities(ctx context.Context) ([]string, error)
}

// BatchOperations defines batch operations interface
type BatchOperations interface {
	BatchUpload(ctx context.Context, requests []*types.UploadRequest) ([]*types.UploadResponse, error)
	BatchDelete(ctx context.Context, paths []string) error
	BatchMove(ctx context.Context, moves []FileMove) error
}

// FileMove defines a file move operation
type FileMove struct {
	SourcePath string
	TargetPath string
}

// SearchOptions defines search options
type SearchOptions struct {
	Query      string
	Prefix     string
	MaxResults int
	SortBy     string
	SortOrder  string
}

// SearchResult defines search result
type SearchResult struct {
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	ContentType  string                 `json:"content_type"`
	LastModified string                 `json:"last_modified"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Searchable defines search interface
type Searchable interface {
	SearchFiles(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error)
	SearchByMetadata(ctx context.Context, key, value string) ([]*SearchResult, error)
}

// PresignedURL defines presigned URL operations
type PresignedURL interface {
	GenerateUploadURL(ctx context.Context, path, contentType string, expiresIn time.Duration) (string, error)
	GenerateDownloadURL(ctx context.Context, path string, expiresIn time.Duration) (string, error)
}

// Webhook defines webhook operations
type Webhook interface {
	RegisterUploadWebhook(ctx context.Context, url string, events []string) error
	UnregisterWebhook(ctx context.Context, url string) error
	TriggerWebhook(ctx context.Context, event string, data map[string]interface{}) error
}

// AdvancedStorage combines all storage interfaces
type AdvancedStorage interface {
	FileStorage
	SnapshotStorage
	StorageStats
	HealthChecker
	BatchOperations
	Searchable
	PresignedURL
	Webhook
}
