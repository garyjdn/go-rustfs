package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/garyjdn/go-rustfs/types"
)

// ValidateFile validates a file based on size and type
func ValidateFile(file io.Reader, filename string, maxSize int64, allowedTypes []string) (*types.FileValidationResult, error) {
	// Read file to get size and content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return &types.FileValidationResult{
			Valid:   false,
			Message: "Failed to read file",
			Code:    "READ_ERROR",
		}, err
	}

	// Check file size
	if maxSize > 0 && int64(n) > maxSize {
		return &types.FileValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("File size %d exceeds maximum allowed size %d", n, maxSize),
			Code:    "FILE_TOO_LARGE",
		}, nil
	}

	// Detect content type
	contentType := http.DetectContentType(buffer)
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}

	// Check if content type is allowed
	if !isAllowedType(contentType, allowedTypes) {
		return &types.FileValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Content type %s is not allowed", contentType),
			Code:    "INVALID_FILE_TYPE",
		}, nil
	}

	return &types.FileValidationResult{
		Valid:   true,
		Message: "File is valid",
		Code:    "VALID",
	}, nil
}

// GenerateFilePath generates a unique file path
func GenerateFilePath(filename string, prefix string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Format("2006/01/02")

	return fmt.Sprintf("%s/%s/%s_%d%s",
		prefix,
		timestamp,
		name,
		time.Now().Unix(),
		ext)
}

// GenerateChecksum generates checksum for file content
func GenerateChecksum(file io.Reader, algorithm string) (string, error) {
	var h hash.Hash

	switch strings.ToLower(algorithm) {
	case "md5":
		h = md5.New()
	case "sha256":
		h = sha256.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetFileInfo extracts file information
func GetFileInfo(file io.Reader, filename string) (*types.FileInfo, error) {
	// Create a temporary buffer to calculate size and checksum
	buffer := make([]byte, 0)
	tempFile := &tempBuffer{buffer: &buffer}

	size, err := io.Copy(tempFile, file)
	if err != nil {
		return nil, err
	}

	// Reset reader for checksum calculation
	reader := io.NopCloser(strings.NewReader(string(*tempFile.buffer)))
	checksum, err := GenerateChecksum(reader, "md5")
	if err != nil {
		return nil, err
	}

	// Detect content type
	contentType := http.DetectContentType((*tempFile.buffer)[:min(512, len(*tempFile.buffer))])
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(filename))
	}

	return &types.FileInfo{
		Path:         filename,
		Size:         size,
		ContentType:  contentType,
		ETag:         checksum,
		LastModified: time.Now(),
	}, nil
}

// IsImageType checks if the content type is an image
func IsImageType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// IsDocumentType checks if the content type is a document
func IsDocumentType(contentType string) bool {
	documentTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"text/plain",
		"text/csv",
	}

	for _, docType := range documentTypes {
		if contentType == docType {
			return true
		}
	}
	return false
}

// SanitizeFilename sanitizes a filename for safe storage
func SanitizeFilename(filename string) string {
	// Remove path separators
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")

	// Remove special characters
	specialChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range specialChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// Trim whitespace and dots
	filename = strings.TrimSpace(filename)
	filename = strings.Trim(filename, ".")

	// Ensure filename is not empty
	if filename == "" {
		filename = fmt.Sprintf("file_%d", time.Now().Unix())
	}

	return filename
}

// GetFileExtension returns the file extension in lowercase
func GetFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// IsAllowedType checks if content type is in allowed list
func isAllowedType(contentType string, allowedTypes []string) bool {
	for _, allowedType := range allowedTypes {
		if matchContentType(allowedType, contentType) {
			return true
		}
	}
	return false
}

// matchContentType checks if content type matches pattern
func matchContentType(pattern, contentType string) bool {
	// Exact match
	if pattern == contentType {
		return true
	}

	// Wildcard match (e.g., "image/*")
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(contentType, prefix+"/")
	}

	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// tempBuffer is a writer that appends to a byte slice
type tempBuffer struct {
	buffer *[]byte
}

func (tb *tempBuffer) Write(p []byte) (n int, err error) {
	*tb.buffer = append(*tb.buffer, p...)
	return len(p), nil
}

// CalculateUploadProgress calculates upload progress
func CalculateUploadProgress(bytesTransferred, totalBytes int64, startTime time.Time) *types.UploadProgress {
	if totalBytes == 0 {
		return &types.UploadProgress{
			BytesTransferred: bytesTransferred,
			TotalBytes:       totalBytes,
			Percentage:       0,
		}
	}

	percentage := float64(bytesTransferred) / float64(totalBytes) * 100

	// Calculate speed and ETA
	elapsed := time.Since(startTime).Seconds()
	var speed int64
	var eta int64

	if elapsed > 0 {
		speed = int64(float64(bytesTransferred) / elapsed)
		remainingBytes := totalBytes - bytesTransferred
		if speed > 0 {
			eta = remainingBytes / speed
		}
	}

	return &types.UploadProgress{
		BytesTransferred: bytesTransferred,
		TotalBytes:       totalBytes,
		Percentage:       percentage,
		Speed:            speed,
		ETA:              eta,
	}
}

// FormatFileSize formats file size in human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp-1])
}

// IsValidFilename checks if filename is valid for storage
func IsValidFilename(filename string) bool {
	if filename == "" {
		return false
	}

	// Check for invalid characters
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// Check for reserved names (Windows)
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	baseName = strings.ToUpper(baseName)

	for _, reserved := range reservedNames {
		if baseName == reserved {
			return false
		}
	}

	return true
}

// GenerateUniqueFilename generates a unique filename by adding timestamp if needed
func GenerateUniqueFilename(basePath string) string {
	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext)
	timestamp := time.Now().Unix()

	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}
