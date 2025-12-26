# RustFS Shared Module

## Overview

RustFS shared module provides a unified interface for file storage operations with comprehensive audit logging integration. This module follows SOLID principles and clean architecture patterns to ensure reusability across multiple services.

## Features

- **Reusable Interface**: Can be used by multiple services with consistent API
- **Audit Integration**: All file operations are automatically audited using the existing audit system
- **Error Handling**: Robust error handling with retry mechanisms
- **File Validation**: Built-in file size and type validation
- **Configuration**: Environment-based configuration management
- **Testing Support**: Mock implementations for unit testing

## Architecture

```
backend/shared/rustfs/
├── client/           # Client implementations
├── config/           # Configuration management
├── types/            # Type definitions
├── audit/            # Audit integration
└── utils/            # Utility functions
```

## Quick Start

### Installation

Add to your service's `go.mod`:

```go
require github.com/garyjdn/go-rustfs v1.0.0

// For local development
replace github.com/garyjdn/go-rustfs => ../../shared/rustfs
```

### Basic Usage

```go
import "github.com/garyjdn/go-rustfs"

// Initialize configuration
config := rustfs.LoadConfig()

// Initialize audit logger
auditLogger := audit.NewAuditLogger("my-service", 
    &audit.ConsoleAuditLogger{},
)

// Create client
client := rustfs.NewAuditableRustFSClient(config, auditLogger, "my-service")

// Upload file with audit
req := &rustfs.UploadRequest{
    File:        file,
    Filename:    "snapshot.png",
    ContentType: "image/png",
    BucketPath:  "snapshots/2024/01/01",
}
result, err := client.UploadFileWithAudit(ctx, req, userID)
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RUSTFS_BASE_URL` | RustFS service URL | `http://localhost:8080` |
| `RUSTFS_API_KEY` | API key for authentication | - |
| `RUSTFS_BUCKET_NAME` | Default bucket name | `default` |
| `RUSTFS_TIMEOUT` | Request timeout | `30s` |
| `RUSTFS_RETRY_COUNT` | Number of retry attempts | `3` |
| `RUSTFS_MAX_FILE_SIZE` | Maximum file size in bytes | `104857600` (100MB) |
| `RUSTFS_ALLOWED_TYPES` | Allowed MIME types | `image/*` |
| `RUSTFS_ENABLE_AUDIT` | Enable audit logging | `true` |
| `RUSTFS_AUDIT_SERVICE` | Service name for audit | `rustfs-client` |

### Configuration Struct

```go
type RustFSConfig struct {
    BaseURL    string        `json:"base_url" env:"RUSTFS_BASE_URL"`
    APIKey     string        `json:"api_key" env:"RUSTFS_API_KEY"`
    BucketName string        `json:"bucket_name" env:"RUSTFS_BUCKET_NAME"`
    Timeout    time.Duration `json:"timeout" env:"RUSTFS_TIMEOUT"`
    RetryCount int           `json:"retry_count" env:"RUSTFS_RETRY_COUNT"`
    MaxFileSize int64        `json:"max_file_size" env:"RUSTFS_MAX_FILE_SIZE"`
    AllowedTypes []string    `json:"allowed_types" env:"RUSTFS_ALLOWED_TYPES"`
    EnableAudit bool          `json:"enable_audit" env:"RUSTFS_ENABLE_AUDIT"`
    AuditService string       `json:"audit_service" env:"RUSTFS_AUDIT_SERVICE"`
}
```

## API Reference

### Interfaces

#### FileStorage

Core interface for file operations:

```go
type FileStorage interface {
    UploadFile(ctx context.Context, req *UploadRequest) (*UploadResponse, error)
    DeleteFile(ctx context.Context, path string) error
    GetFileURL(path string) string
    GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
}
```

#### AuditableStorage

Extended interface with audit capabilities:

```go
type AuditableStorage interface {
    FileStorage
    UploadFileWithAudit(ctx context.Context, req *UploadRequest, userID string) (*UploadResponse, error)
    DeleteFileWithAudit(ctx context.Context, path, userID string) error
}
```

#### SnapshotStorage

Specialized interface for snapshot operations:

```go
type SnapshotStorage interface {
    UploadSnapshot(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error)
    DeleteSnapshot(ctx context.Context, path string) error
    GetSnapshotURL(path string) string
}
```

### Request/Response Types

#### UploadRequest

```go
type UploadRequest struct {
    File        io.Reader     `json:"-"`
    Filename    string        `json:"filename"`
    ContentType string        `json:"content_type"`
    FileSize    int64         `json:"file_size"`
    BucketPath  string        `json:"bucket_path"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

#### UploadResponse

```go
type UploadResponse struct {
    Path         string                    `json:"path"`
    URL          string                    `json:"url"`
    Size         int64                     `json:"size"`
    ContentType  string                    `json:"content_type"`
    ETag         string                    `json:"etag"`
    LastModified time.Time                 `json:"last_modified"`
    Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}
```

#### FileInfo

```go
type FileInfo struct {
    Path         string                    `json:"path"`
    Size         int64                     `json:"size"`
    ContentType  string                    `json:"content_type"`
    ETag         string                    `json:"etag"`
    LastModified time.Time                 `json:"last_modified"`
    Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}
```

## Audit Integration

The module automatically logs the following audit events:

- `file_uploaded` - When a file is successfully uploaded
- `file_deleted` - When a file is deleted
- `file_accessed` - When file information is accessed
- `storage_error` - When storage operations fail
- `storage_quota_exceeded` - When file size limits are exceeded
- `storage_access_denied` - When access is denied

### Audit Event Structure

```go
{
    "id": "uuid",
    "timestamp": "2024-01-01T12:00:00Z",
    "service": "site-service",
    "event_type": "file_uploaded",
    "user_id": "user-uuid",
    "resource": "file",
    "resource_id": "snapshots/2024/01/01/file.png",
    "success": true,
    "metadata": {
        "file_name": "file.png",
        "file_size": 1024000,
        "content_type": "image/png",
        "upload_time": "2024-01-01T12:00:00Z"
    }
}
```

## Error Handling

The module provides comprehensive error handling with proper error types:

```go
type RustFSError struct {
    *apperror.AppError
    ErrorCode    string
    Details      map[string]interface{}
}
```

Common error codes:
- `INVALID_FILE_TYPE` - File type not allowed
- `FILE_TOO_LARGE` - File exceeds size limit
- `UPLOAD_FAILED` - Upload operation failed
- `DELETE_FAILED` - Delete operation failed
- `NOT_FOUND` - File not found
- `ACCESS_DENIED` - Access to file denied

## Examples

### Site Service Integration

```go
// In site page usecase
type sitePageUsecase struct {
    // ... existing fields
    rustfsClient rustfs.AuditableStorage
}

func (u *sitePageUsecase) UpdateBaseline(ctx context.Context, userID, siteID, pageID string, baselineHTML, snapshotData string) (*entity.SitePage, error) {
    // Convert base64 to file
    snapshotFile, header, err := u.convertBase64ToFile(snapshotData)
    if err != nil {
        return nil, apperror.NewAppError(400, "Invalid snapshot data", err)
    }
    defer snapshotFile.Close()
    
    // Upload with audit
    uploadReq := &rustfs.UploadRequest{
        File:        snapshotFile,
        Filename:    header.Filename,
        ContentType: header.Header.Get("Content-Type"),
        BucketPath:  fmt.Sprintf("snapshots/%s/%s", siteID, pageID),
    }
    
    result, err := u.rustfsClient.UploadFileWithAudit(ctx, uploadReq, userID)
    if err != nil {
        return nil, apperror.NewAppError(500, "Failed to upload snapshot", err)
    }
    
    // Update database
    return u.updatePageBaseline(ctx, pageID, baselineHTML, result.Path)
}
```

### HTTP Handler Integration

```go
func (h *HTTPHandler) UpdateBaseline(w http.ResponseWriter, r *http.Request) {
    userID := r.Header.Get("X-User-ID")
    siteID := chi.URLParam(r, "id")
    pageID := chi.URLParam(r, "page_id")

    // Parse multipart form
    if err := r.ParseMultipartForm(32 << 20); err != nil {
        httputils.WriteErrorResponse(w, apperror.NewAppError(400, "Failed to parse form", err))
        return
    }

    // Get file from form
    file, header, err := r.FormFile("snapshot")
    if err != nil {
        httputils.WriteErrorResponse(w, apperror.NewAppError(400, "Snapshot file required", err))
        return
    }
    defer file.Close()

    baselineHTML := r.FormValue("baseline_html")
    
    // Update with file upload
    page, err := h.sitePageUsecase.UpdateBaselineWithFile(ctx, userID, siteID, pageID, baselineHTML, file, header)
    if err != nil {
        httputils.WriteErrorResponse(w, err)
        return
    }

    httputils.WriteSuccessResponse(w, page)
}
```

## Testing

### Mock Client

```go
func TestSitePageUsecase_UpdateBaseline(t *testing.T) {
    // Create mock client
    mockClient := &rustfs.MockRustFSClient{
        UploadFunc: func(ctx context.Context, req *rustfs.UploadRequest) (*rustfs.UploadResponse, error) {
            return &rustfs.UploadResponse{
                Path: "snapshots/test/file.png",
                URL:  "http://rustfs:8080/bucket/snapshots/test/file.png",
                Size: 1024000,
            }, nil
        },
    }
    
    // Create usecase with mock
    usecase := &sitePageUsecase{
        rustfsClient: mockClient,
        // ... other dependencies
    }
    
    // Test
    result, err := usecase.UpdateBaseline(ctx, userID, siteID, pageID, baselineHTML, snapshotData)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Contributing

1. Follow the existing code patterns and SOLID principles
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Ensure audit logging is implemented for all operations

## License

MIT License - see LICENSE file for details.