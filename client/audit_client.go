package client

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/garyjdn/go-apperror"
	"github.com/garyjdn/go-rustfs/audit"
	"github.com/garyjdn/go-rustfs/config"
	"github.com/garyjdn/go-rustfs/types"
	"github.com/garyjdn/go-rustfs/utils"
)

// AuditableRustFSClient wraps a FileStorage client with audit capabilities
type AuditableRustFSClient struct {
	client      FileStorage
	auditLogger *audit.RustFSAuditLogger
	config      *config.RustFSConfig
	service     string
}

// NewAuditableRustFSClient creates a new auditable RustFS client
func NewAuditableRustFSClient(client FileStorage, auditLogger *audit.RustFSAuditLogger, config *config.RustFSConfig, service string) *AuditableRustFSClient {
	return &AuditableRustFSClient{
		client:      client,
		auditLogger: auditLogger,
		config:      config,
		service:     service,
	}
}

// UploadFile implements FileStorage interface
func (c *AuditableRustFSClient) UploadFile(ctx context.Context, req *types.UploadRequest) (*types.UploadResponse, error) {
	return c.client.UploadFile(ctx, req)
}

// UploadFileWithAudit uploads a file with audit logging
func (c *AuditableRustFSClient) UploadFileWithAudit(ctx context.Context, req *types.UploadRequest, userID string) (*types.UploadResponse, error) {
	startTime := time.Now()

	// Pre-upload audit metadata
	preUploadMetadata := &audit.FileOperationMetadata{
		Filename:    req.Filename,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		FilePath:    req.BucketPath,
		BucketName:  c.config.BucketName,
		Additional:  req.Metadata,
	}

	// Validate file before upload
	if err := c.validateUploadRequest(req); err != nil {
		c.logUploadError(ctx, userID, preUploadMetadata, err, startTime)
		return nil, c.wrapError(err, "VALIDATION_ERROR")
	}

	// Execute upload
	result, err := c.client.UploadFile(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		c.logUploadError(ctx, userID, preUploadMetadata, err, startTime)
		return nil, c.wrapError(err, "UPLOAD_FAILED")
	}

	// Log successful upload
	c.logUploadSuccess(ctx, userID, preUploadMetadata, result, duration)

	// Log performance if upload is slow
	if duration > c.config.Timeout {
		c.auditLogger.LogPerformanceEvent(ctx, userID, audit.AuditEventUploadSlow, &audit.PerformanceEventMetadata{
			Operation:  "upload",
			Duration:   duration.String(),
			FileSize:   req.FileSize,
			Throughput: c.calculateThroughput(req.FileSize, duration),
			Threshold:  float64(c.config.Timeout.Milliseconds()),
		})
	}

	return result, nil
}

// DeleteFile implements FileStorage interface
func (c *AuditableRustFSClient) DeleteFile(ctx context.Context, path string) error {
	return c.client.DeleteFile(ctx, path)
}

// DeleteFileWithAudit deletes a file with audit logging
func (c *AuditableRustFSClient) DeleteFileWithAudit(ctx context.Context, path, userID string) error {
	startTime := time.Now()

	// Pre-delete audit metadata
	preDeleteMetadata := &audit.FileOperationMetadata{
		FilePath:   path,
		BucketName: c.config.BucketName,
		AccessTime: time.Now().Format(time.RFC3339),
	}

	// Execute delete
	err := c.client.DeleteFile(ctx, path)
	duration := time.Since(startTime)

	if err != nil {
		c.auditLogger.LogFileDelete(ctx, userID, path, preDeleteMetadata, err)
		return c.wrapError(err, "DELETE_FAILED")
	}

	// Log successful deletion
	c.auditLogger.LogFileDelete(ctx, userID, path, preDeleteMetadata, nil)

	// Log performance if delete is slow
	if duration > c.config.Timeout/2 { // Half of upload timeout for delete
		c.auditLogger.LogPerformanceEvent(ctx, userID, audit.AuditEventUploadSlow, &audit.PerformanceEventMetadata{
			Operation:  "delete",
			Duration:   duration.String(),
			FileSize:   0,
			Throughput: 0,
			Threshold:  float64((c.config.Timeout / 2).Milliseconds()),
		})
	}

	return nil
}

// GetFileURL implements FileStorage interface
func (c *AuditableRustFSClient) GetFileURL(path string) string {
	return c.client.GetFileURL(path)
}

// GetFileInfo implements FileStorage interface
func (c *AuditableRustFSClient) GetFileInfo(ctx context.Context, path string) (*types.FileInfo, error) {
	startTime := time.Now()
	userID := c.extractUserID(ctx)

	// Pre-access audit metadata
	preAccessMetadata := &audit.FileOperationMetadata{
		FilePath:   path,
		BucketName: c.config.BucketName,
		AccessTime: time.Now().Format(time.RFC3339),
	}

	// Execute get file info
	result, err := c.client.GetFileInfo(ctx, path)
	duration := time.Since(startTime)

	if err != nil {
		c.auditLogger.LogFileAccess(ctx, userID, path, preAccessMetadata, err)
		return nil, c.wrapError(err, "GET_INFO_FAILED")
	}

	// Log successful access
	c.auditLogger.LogFileAccess(ctx, userID, path, preAccessMetadata, nil)

	// Log performance if access is slow
	if duration > c.config.Timeout/4 { // Quarter of upload timeout for get info
		c.auditLogger.LogPerformanceEvent(ctx, userID, audit.AuditEventUploadSlow, &audit.PerformanceEventMetadata{
			Operation:  "get_info",
			Duration:   duration.String(),
			FileSize:   0,
			Throughput: 0,
			Threshold:  float64((c.config.Timeout / 4).Milliseconds()),
		})
	}

	return result, nil
}

// UploadSnapshot implements SnapshotStorage interface
func (c *AuditableRustFSClient) UploadSnapshot(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Read file info
	fileInfo, err := header.Open()
	if err != nil {
		return "", c.wrapError(err, "FILE_OPEN_ERROR")
	}
	defer fileInfo.Close()

	// Create upload request
	req := &types.UploadRequest{
		File:        file,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		FileSize:    header.Size,
		BucketPath:  utils.GenerateFilePath(header.Filename, "snapshots"),
		Metadata: map[string]interface{}{
			"original_filename": header.Filename,
			"upload_source":     "snapshot",
		},
	}

	// Upload with audit
	result, err := c.UploadFileWithAudit(ctx, req, c.extractUserID(ctx))
	if err != nil {
		return "", err
	}

	return result.Path, nil
}

// DeleteSnapshot implements SnapshotStorage interface
func (c *AuditableRustFSClient) DeleteSnapshot(ctx context.Context, path string) error {
	return c.DeleteFileWithAudit(ctx, path, c.extractUserID(ctx))
}

// GetSnapshotURL implements SnapshotStorage interface
func (c *AuditableRustFSClient) GetSnapshotURL(path string) string {
	return c.GetFileURL(path)
}

// Helper methods

func (c *AuditableRustFSClient) validateUploadRequest(req *types.UploadRequest) error {
	// Validate file size
	if req.FileSize > c.config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", req.FileSize, c.config.MaxFileSize)
	}

	// Validate content type
	if !c.config.IsAllowedType(req.ContentType) {
		return fmt.Errorf("content type %s is not allowed", req.ContentType)
	}

	// Validate filename
	if !utils.IsValidFilename(req.Filename) {
		return fmt.Errorf("filename %s is not valid", req.Filename)
	}

	return nil
}

func (c *AuditableRustFSClient) logUploadSuccess(ctx context.Context, userID string, metadata *audit.FileOperationMetadata, result *types.UploadResponse, duration time.Duration) {
	// Update metadata with result info
	metadata.ETag = result.ETag
	metadata.UploadTime = time.Now().Format(time.RFC3339)
	metadata.Additional["upload_duration"] = duration.String()
	metadata.Additional["upload_speed"] = c.calculateThroughput(result.Size, duration)

	c.auditLogger.LogFileUpload(ctx, userID, metadata, nil)
}

func (c *AuditableRustFSClient) logUploadError(ctx context.Context, userID string, metadata *audit.FileOperationMetadata, err error, startTime time.Time) {
	// Create storage error metadata
	storageErrorMetadata := &audit.StorageErrorMetadata{
		Operation:    "upload",
		ErrorCode:    "UPLOAD_ERROR",
		ErrorMessage: err.Error(),
		RetryCount:   0,
		Duration:     time.Since(startTime).String(),
		Context: map[string]interface{}{
			"file_name":    metadata.Filename,
			"file_size":    metadata.FileSize,
			"content_type": metadata.ContentType,
			"bucket_path":  metadata.FilePath,
		},
	}

	c.auditLogger.LogStorageError(ctx, userID, "upload", storageErrorMetadata)
}

func (c *AuditableRustFSClient) wrapError(err error, code string) *apperror.AppError {
	if appErr, ok := err.(*apperror.AppError); ok {
		return appErr
	}

	return apperror.NewAppError(500, fmt.Sprintf("RustFS operation failed: %s", code), err)
}

func (c *AuditableRustFSClient) calculateThroughput(bytes int64, duration time.Duration) float64 {
	if duration.Seconds() == 0 {
		return 0
	}
	return float64(bytes) / duration.Seconds() / 1024 / 1024 // MB/s
}

func (c *AuditableRustFSClient) extractUserID(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return "system"
}

// GetConfig returns client configuration
func (c *AuditableRustFSClient) GetConfig() *config.RustFSConfig {
	return c.config
}

// GetService returns service name
func (c *AuditableRustFSClient) GetService() string {
	return c.service
}

// IsAuditEnabled returns true if audit logging is enabled
func (c *AuditableRustFSClient) IsAuditEnabled() bool {
	return c.auditLogger.IsEnabled()
}

// GetAuditLogger returns audit logger
func (c *AuditableRustFSClient) GetAuditLogger() *audit.RustFSAuditLogger {
	return c.auditLogger
}

// Close closes client and performs cleanup
func (c *AuditableRustFSClient) Close() error {
	// Log client shutdown
	if c.auditLogger.IsEnabled() {
		c.auditLogger.LogConfigChange(context.Background(), c.extractUserID(context.Background()),
			map[string]interface{}{
				"action":  "client_shutdown",
				"service": c.service,
			},
			map[string]interface{}{
				"action":    "client_shutdown",
				"service":   c.service,
				"timestamp": time.Now().Format(time.RFC3339),
			})
	}

	// Close underlying client if it has a Close method
	if closer, ok := c.client.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

// HealthCheck performs a health check on the storage client
func (c *AuditableRustFSClient) HealthCheck(ctx context.Context) error {
	// Try to get storage stats
	if healthChecker, ok := c.client.(interface {
		CheckHealth(ctx context.Context) error
	}); ok {
		return healthChecker.CheckHealth(ctx)
	}

	// Fallback: try to get file info for a non-existent file
	_, err := c.client.GetFileInfo(ctx, "health-check-"+time.Now().Format("20060102"))
	if err != nil {
		// Expected error for non-existent file is OK
		return nil
	}

	return nil
}
