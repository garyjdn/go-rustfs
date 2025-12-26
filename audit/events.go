package audit

import (
	"github.com/garyjdn/go-auditlogger/types"
)

// RustFS-specific audit event types
const (
	// File operations (extend existing data events)
	AuditEventFileUploaded   types.AuditEventType = "file_uploaded"
	AuditEventFileDeleted    types.AuditEventType = "file_deleted"
	AuditEventFileAccessed   types.AuditEventType = "file_accessed"
	AuditEventFileDownloaded types.AuditEventType = "file_downloaded"
	AuditEventFileUpdated    types.AuditEventType = "file_updated"
	AuditEventFileCopied     types.AuditEventType = "file_copied"
	AuditEventFileMoved      types.AuditEventType = "file_moved"

	// Storage-specific events
	AuditEventStorageError         types.AuditEventType = "storage_error"
	AuditEventStorageQuotaExceeded types.AuditEventType = "storage_quota_exceeded"
	AuditEventStorageAccessDenied  types.AuditEventType = "storage_access_denied"
	AuditEventStorageConfigChanged types.AuditEventType = "storage_config_changed"
	AuditEventStorageMaintenance   types.AuditEventType = "storage_maintenance"

	// Security events
	AuditEventMalwareDetected    types.AuditEventType = "malware_detected"
	AuditEventSuspiciousFile     types.AuditEventType = "suspicious_file"
	AuditEventUnauthorizedAccess types.AuditEventType = "unauthorized_access"
	AuditEventDataBreach         types.AuditEventType = "data_breach"

	// Performance events
	AuditEventUploadSlow        types.AuditEventType = "upload_slow"
	AuditEventUploadTimeout     types.AuditEventType = "upload_timeout"
	AuditEventStorageFull       types.AuditEventType = "storage_full"
	AuditEventHighResourceUsage types.AuditEventType = "high_resource_usage"
)

// FileOperationMetadata represents metadata for file operations
type FileOperationMetadata struct {
	Filename     string                 `json:"filename"`
	FileSize     int64                  `json:"file_size"`
	ContentType  string                 `json:"content_type"`
	FilePath     string                 `json:"file_path"`
	BucketName   string                 `json:"bucket_name"`
	UploadTime   string                 `json:"upload_time,omitempty"`
	DownloadTime string                 `json:"download_time,omitempty"`
	AccessTime   string                 `json:"access_time,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	ETag         string                 `json:"etag,omitempty"`
	Checksum     string                 `json:"checksum,omitempty"`
	Additional   map[string]interface{} `json:"additional,omitempty"`
}

// StorageErrorMetadata represents metadata for storage errors
type StorageErrorMetadata struct {
	Operation    string                 `json:"operation"`
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	RetryCount   int                    `json:"retry_count"`
	Duration     string                 `json:"duration"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// SecurityEventMetadata represents metadata for security events
type SecurityEventMetadata struct {
	ThreatType    string                 `json:"threat_type"`
	ThreatLevel   string                 `json:"threat_level"`
	FileSignature string                 `json:"file_signature,omitempty"`
	ScanResult    string                 `json:"scan_result,omitempty"`
	Blocked       bool                   `json:"blocked"`
	Action        string                 `json:"action"`
	Additional    map[string]interface{} `json:"additional,omitempty"`
}

// PerformanceEventMetadata represents metadata for performance events
type PerformanceEventMetadata struct {
	Operation     string                 `json:"operation"`
	Duration      string                 `json:"duration"`
	FileSize      int64                  `json:"file_size"`
	Throughput    float64                `json:"throughput_mbps"`
	Concurrency   int                    `json:"concurrency"`
	ResourceUsage string                 `json:"resource_usage"`
	Threshold     float64                `json:"threshold"`
	Additional    map[string]interface{} `json:"additional,omitempty"`
}

// Update severity mapping for RustFS events
func GetSeverity(eventType types.AuditEventType) types.AuditSeverity {
	switch eventType {
	// File operations
	case AuditEventFileUploaded, AuditEventFileAccessed, AuditEventFileDownloaded, AuditEventFileCopied, AuditEventFileMoved:
		return types.AuditSeverityLow
	case AuditEventFileDeleted, AuditEventFileUpdated:
		return types.AuditSeverityMedium

	// Storage operations
	case AuditEventStorageConfigChanged, AuditEventStorageMaintenance:
		return types.AuditSeverityMedium
	case AuditEventStorageError:
		return types.AuditSeverityHigh
	case AuditEventStorageQuotaExceeded, AuditEventStorageAccessDenied:
		return types.AuditSeverityCritical
	case AuditEventStorageFull:
		return types.AuditSeverityCritical

	// Security events
	case AuditEventSuspiciousFile:
		return types.AuditSeverityHigh
	case AuditEventMalwareDetected, AuditEventUnauthorizedAccess, AuditEventDataBreach:
		return types.AuditSeverityCritical

	// Performance events
	case AuditEventUploadSlow:
		return types.AuditSeverityMedium
	case AuditEventUploadTimeout, AuditEventHighResourceUsage:
		return types.AuditSeverityHigh

	default:
		// Fall back to original severity mapping
		return types.GetSeverity(eventType)
	}
}

// IsSecurityEvent checks if an event type is security-related for RustFS
func IsSecurityEvent(eventType types.AuditEventType) bool {
	switch eventType {
	case AuditEventMalwareDetected, AuditEventSuspiciousFile, AuditEventUnauthorizedAccess, AuditEventDataBreach:
		return true
	default:
		return types.IsSecurityEvent(eventType)
	}
}

// IsPerformanceEvent checks if an event type is performance-related
func IsPerformanceEvent(eventType types.AuditEventType) bool {
	switch eventType {
	case AuditEventUploadSlow, AuditEventUploadTimeout, AuditEventStorageFull, AuditEventHighResourceUsage:
		return true
	default:
		return false
	}
}

// IsFileEvent checks if an event type is file operation-related
func IsFileEvent(eventType types.AuditEventType) bool {
	switch eventType {
	case AuditEventFileUploaded, AuditEventFileDeleted, AuditEventFileAccessed, AuditEventFileDownloaded, AuditEventFileUpdated, AuditEventFileCopied, AuditEventFileMoved:
		return true
	default:
		return false
	}
}

// IsStorageEvent checks if an event type is storage-related
func IsStorageEvent(eventType types.AuditEventType) bool {
	switch eventType {
	case AuditEventStorageError, AuditEventStorageQuotaExceeded, AuditEventStorageAccessDenied, AuditEventStorageConfigChanged, AuditEventStorageMaintenance, AuditEventStorageFull:
		return true
	default:
		return false
	}
}
