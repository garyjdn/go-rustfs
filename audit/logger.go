package audit

import (
	"context"
	"fmt"
	"time"

	audittypes "github.com/garyjdn/go-auditlogger/types"
)

// RustFSAuditLogger wraps general audit logger with RustFS-specific functionality
type RustFSAuditLogger struct {
	auditLogger audittypes.AuditLogger
	service     string
	config      map[string]interface{}
}

// NewRustFSAuditLogger creates a new RustFS-specific audit logger
func NewRustFSAuditLogger(service string, auditLogger audittypes.AuditLogger, config map[string]interface{}) *RustFSAuditLogger {
	return &RustFSAuditLogger{
		auditLogger: auditLogger,
		service:     service,
		config:      config,
	}
}

// LogFileUpload logs a file upload event
func (l *RustFSAuditLogger) LogFileUpload(ctx context.Context, userID string, metadata *FileOperationMetadata, err error) {
	eventType := AuditEventFileUploaded
	success := err == nil

	auditMetadata := l.buildFileMetadata(metadata)
	if err != nil {
		eventType = AuditEventStorageError
		auditMetadata["error"] = err.Error()
		auditMetadata["error_type"] = "upload_failed"
	}

	event := &audittypes.AuditEvent{
		EventType:  eventType,
		UserID:     userID,
		Resource:   "file",
		ResourceID: metadata.FilePath,
		Success:    success,
		Reason:     l.getReason(success, err),
		Metadata:   auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogFileDelete logs a file deletion event
func (l *RustFSAuditLogger) LogFileDelete(ctx context.Context, userID, filePath string, metadata *FileOperationMetadata, err error) {
	eventType := AuditEventFileDeleted
	success := err == nil

	auditMetadata := l.buildFileMetadata(metadata)
	if err != nil {
		eventType = AuditEventStorageError
		auditMetadata["error"] = err.Error()
		auditMetadata["error_type"] = "delete_failed"
	}

	event := &audittypes.AuditEvent{
		EventType:  eventType,
		UserID:     userID,
		Resource:   "file",
		ResourceID: filePath,
		Success:    success,
		Reason:     l.getReason(success, err),
		Metadata:   auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogFileAccess logs a file access event
func (l *RustFSAuditLogger) LogFileAccess(ctx context.Context, userID, filePath string, metadata *FileOperationMetadata, err error) {
	eventType := AuditEventFileAccessed
	success := err == nil

	auditMetadata := l.buildFileMetadata(metadata)
	if err != nil {
		eventType = AuditEventStorageAccessDenied
		auditMetadata["error"] = err.Error()
		auditMetadata["error_type"] = "access_denied"
	}

	event := &audittypes.AuditEvent{
		EventType:  eventType,
		UserID:     userID,
		Resource:   "file",
		ResourceID: filePath,
		Success:    success,
		Reason:     l.getReason(success, err),
		Metadata:   auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogFileDownload logs a file download event
func (l *RustFSAuditLogger) LogFileDownload(ctx context.Context, userID, filePath string, metadata *FileOperationMetadata, err error) {
	eventType := AuditEventFileDownloaded
	success := err == nil

	auditMetadata := l.buildFileMetadata(metadata)
	if err != nil {
		eventType = AuditEventStorageError
		auditMetadata["error"] = err.Error()
		auditMetadata["error_type"] = "download_failed"
	}

	event := &audittypes.AuditEvent{
		EventType:  eventType,
		UserID:     userID,
		Resource:   "file",
		ResourceID: filePath,
		Success:    success,
		Reason:     l.getReason(success, err),
		Metadata:   auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogStorageError logs a storage error event
func (l *RustFSAuditLogger) LogStorageError(ctx context.Context, userID, operation string, metadata *StorageErrorMetadata) {
	auditMetadata := map[string]interface{}{
		"operation":     metadata.Operation,
		"error_code":    metadata.ErrorCode,
		"error_message": metadata.ErrorMessage,
		"retry_count":   metadata.RetryCount,
		"duration":      metadata.Duration,
		"service":       l.service,
	}

	// Add context if available
	if metadata.Context != nil {
		for k, v := range metadata.Context {
			auditMetadata[k] = v
		}
	}

	// Add config if available
	if l.config != nil {
		auditMetadata["config"] = l.config
	}

	event := &audittypes.AuditEvent{
		EventType: AuditEventStorageError,
		UserID:    userID,
		Resource:  "storage",
		Success:   false,
		Reason:    metadata.ErrorMessage,
		Metadata:  auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogSecurityEvent logs a security-related event
func (l *RustFSAuditLogger) LogSecurityEvent(ctx context.Context, userID string, eventType audittypes.AuditEventType, metadata *SecurityEventMetadata) {
	auditMetadata := map[string]interface{}{
		"threat_type":  metadata.ThreatType,
		"threat_level": metadata.ThreatLevel,
		"blocked":      metadata.Blocked,
		"action":       metadata.Action,
		"service":      l.service,
	}

	if metadata.FileSignature != "" {
		auditMetadata["file_signature"] = metadata.FileSignature
	}

	if metadata.ScanResult != "" {
		auditMetadata["scan_result"] = metadata.ScanResult
	}

	// Add additional metadata if available
	if metadata.Additional != nil {
		for k, v := range metadata.Additional {
			auditMetadata[k] = v
		}
	}

	event := &audittypes.AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Resource:  "file",
		Success:   !metadata.Blocked,
		Reason:    fmt.Sprintf("Security event: %s", metadata.ThreatType),
		Metadata:  auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogPerformanceEvent logs a performance-related event
func (l *RustFSAuditLogger) LogPerformanceEvent(ctx context.Context, userID string, eventType audittypes.AuditEventType, metadata *PerformanceEventMetadata) {
	auditMetadata := map[string]interface{}{
		"operation":      metadata.Operation,
		"duration":       metadata.Duration,
		"file_size":      metadata.FileSize,
		"throughput":     metadata.Throughput,
		"concurrency":    metadata.Concurrency,
		"resource_usage": metadata.ResourceUsage,
		"threshold":      metadata.Threshold,
		"service":        l.service,
	}

	// Add additional metadata if available
	if metadata.Additional != nil {
		for k, v := range metadata.Additional {
			auditMetadata[k] = v
		}
	}

	event := &audittypes.AuditEvent{
		EventType: eventType,
		UserID:    userID,
		Resource:  "storage",
		Success:   true,
		Metadata:  auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogQuotaExceeded logs a quota exceeded event
func (l *RustFSAuditLogger) LogQuotaExceeded(ctx context.Context, userID string, currentUsage, quotaLimit int64, metadata *FileOperationMetadata) {
	auditMetadata := l.buildFileMetadata(metadata)
	auditMetadata["current_usage"] = currentUsage
	auditMetadata["quota_limit"] = quotaLimit
	auditMetadata["percentage_used"] = float64(currentUsage) / float64(quotaLimit) * 100

	event := &audittypes.AuditEvent{
		EventType: AuditEventStorageQuotaExceeded,
		UserID:    userID,
		Resource:  "storage",
		Success:   false,
		Reason:    "Storage quota exceeded",
		Metadata:  auditMetadata,
	}

	l.logEvent(ctx, event)
}

// LogConfigChange logs a configuration change event
func (l *RustFSAuditLogger) LogConfigChange(ctx context.Context, userID string, oldConfig, newConfig map[string]interface{}) {
	auditMetadata := map[string]interface{}{
		"old_config": oldConfig,
		"new_config": newConfig,
		"service":    l.service,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	event := &audittypes.AuditEvent{
		EventType: AuditEventStorageConfigChanged,
		UserID:    userID,
		Resource:  "config",
		Success:   true,
		Reason:    "Configuration updated",
		Metadata:  auditMetadata,
	}

	l.logEvent(ctx, event)
}

// Helper methods

func (l *RustFSAuditLogger) logEvent(ctx context.Context, event *audittypes.AuditEvent) {
	if l.auditLogger != nil {
		l.auditLogger.LogEvent(ctx, event)
	}
}

func (l *RustFSAuditLogger) buildFileMetadata(metadata *FileOperationMetadata) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}

	auditMetadata := map[string]interface{}{
		"service": l.service,
	}

	if metadata.Filename != "" {
		auditMetadata["file_name"] = metadata.Filename
	}

	if metadata.FileSize > 0 {
		auditMetadata["file_size"] = metadata.FileSize
	}

	if metadata.ContentType != "" {
		auditMetadata["content_type"] = metadata.ContentType
	}

	if metadata.FilePath != "" {
		auditMetadata["file_path"] = metadata.FilePath
	}

	if metadata.BucketName != "" {
		auditMetadata["bucket_name"] = metadata.BucketName
	}

	if metadata.UploadTime != "" {
		auditMetadata["upload_time"] = metadata.UploadTime
	}

	if metadata.DownloadTime != "" {
		auditMetadata["download_time"] = metadata.DownloadTime
	}

	if metadata.AccessTime != "" {
		auditMetadata["access_time"] = metadata.AccessTime
	}

	if metadata.UserAgent != "" {
		auditMetadata["user_agent"] = metadata.UserAgent
	}

	if metadata.IPAddress != "" {
		auditMetadata["ip_address"] = metadata.IPAddress
	}

	if metadata.ETag != "" {
		auditMetadata["etag"] = metadata.ETag
	}

	if metadata.Checksum != "" {
		auditMetadata["checksum"] = metadata.Checksum
	}

	// Add additional metadata if available
	if metadata.Additional != nil {
		for k, v := range metadata.Additional {
			auditMetadata[k] = v
		}
	}

	// Add config if available
	if l.config != nil {
		auditMetadata["config"] = l.config
	}

	return auditMetadata
}

func (l *RustFSAuditLogger) getReason(success bool, err error) string {
	if success {
		return "Operation completed successfully"
	}

	if err != nil {
		return err.Error()
	}

	return "Operation failed"
}

// GetAuditLogger returns the underlying audit logger
func (l *RustFSAuditLogger) GetAuditLogger() audittypes.AuditLogger {
	return l.auditLogger
}

// GetService returns the service name
func (l *RustFSAuditLogger) GetService() string {
	return l.service
}

// IsEnabled returns true if audit logging is enabled
func (l *RustFSAuditLogger) IsEnabled() bool {
	return l.auditLogger != nil
}
