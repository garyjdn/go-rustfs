package client

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"sync"
	"time"

	"github.com/garyjdn/go-rustfs/types"
)

// MockRustFSClient is a mock implementation of FileStorage interface for testing
type MockRustFSClient struct {
	files      map[string]*types.FileInfo
	uploads    []*types.UploadResponse
	deletes    []string
	mu         sync.RWMutex
	shouldFail bool
	failError  error
}

// NewMockRustFSClient creates a new mock RustFS client
func NewMockRustFSClient() *MockRustFSClient {
	return &MockRustFSClient{
		files:   make(map[string]*types.FileInfo),
		uploads: make([]*types.UploadResponse, 0),
		deletes: make([]string, 0),
	}
}

// SetFailureMode sets the mock client to fail on next operation
func (m *MockRustFSClient) SetFailureMode(shouldFail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failError = err
}

// UploadFile uploads a file to mock storage
func (m *MockRustFSClient) UploadFile(ctx context.Context, req *types.UploadRequest) (*types.UploadResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return nil, m.failError
	}

	// Simulate upload delay
	time.Sleep(10 * time.Millisecond)

	// Create upload response
	response := &types.UploadResponse{
		Path:     req.BucketPath,
		ETag:     fmt.Sprintf("etag-%d", time.Now().UnixNano()),
		Size:     req.FileSize,
		URL:      fmt.Sprintf("http://mock-storage.com/%s", req.BucketPath),
		Metadata: req.Metadata,
	}

	// Store file info
	fileInfo := &types.FileInfo{
		Path:         req.BucketPath,
		Size:         req.FileSize,
		ContentType:  req.ContentType,
		ETag:         response.ETag,
		LastModified: time.Now(),
		Metadata:     req.Metadata,
	}

	m.files[req.BucketPath] = fileInfo
	m.uploads = append(m.uploads, response)

	return response, nil
}

// DeleteFile deletes a file from mock storage
func (m *MockRustFSClient) DeleteFile(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return m.failError
	}

	// Simulate delete delay
	time.Sleep(5 * time.Millisecond)

	// Remove file if exists
	if _, exists := m.files[path]; exists {
		delete(m.files, path)
	}

	m.deletes = append(m.deletes, path)
	return nil
}

// GetFileURL returns mock URL for a file
func (m *MockRustFSClient) GetFileURL(path string) string {
	return fmt.Sprintf("http://mock-storage.com/%s", path)
}

// GetFileInfo retrieves file information from mock storage
func (m *MockRustFSClient) GetFileInfo(ctx context.Context, path string) (*types.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return nil, m.failError
	}

	// Simulate get info delay
	time.Sleep(2 * time.Millisecond)

	fileInfo, exists := m.files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	return fileInfo, nil
}

// GetFiles returns all files in mock storage
func (m *MockRustFSClient) GetFiles() map[string]*types.FileInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make(map[string]*types.FileInfo)
	for k, v := range m.files {
		files[k] = v
	}
	return files
}

// GetUploads returns all upload responses
func (m *MockRustFSClient) GetUploads() []*types.UploadResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uploads := make([]*types.UploadResponse, len(m.uploads))
	copy(uploads, m.uploads)
	return uploads
}

// GetDeletes returns all deleted paths
func (m *MockRustFSClient) GetDeletes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deletes := make([]string, len(m.deletes))
	copy(deletes, m.deletes)
	return deletes
}

// Reset clears all mock data
func (m *MockRustFSClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files = make(map[string]*types.FileInfo)
	m.uploads = make([]*types.UploadResponse, 0)
	m.deletes = make([]string, 0)
	m.shouldFail = false
	m.failError = nil
}

// UploadSnapshot implements SnapshotStorage interface
func (m *MockRustFSClient) UploadSnapshot(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	// Read file content to get size
	fileInfo, err := header.Open()
	if err != nil {
		return "", err
	}
	defer fileInfo.Close()

	// Read content to ensure file is readable
	_, err = io.ReadAll(fileInfo)
	if err != nil {
		return "", err
	}

	// Create upload request
	req := &types.UploadRequest{
		File:        file,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		FileSize:    header.Size,
		BucketPath:  fmt.Sprintf("snapshots/%s/%s", time.Now().Format("2006/01/02"), header.Filename),
		Metadata: map[string]interface{}{
			"original_filename": header.Filename,
			"upload_source":     "snapshot",
		},
	}

	// Upload file
	response, err := m.UploadFile(ctx, req)
	if err != nil {
		return "", err
	}

	return response.Path, nil
}

// DeleteSnapshot implements SnapshotStorage interface
func (m *MockRustFSClient) DeleteSnapshot(ctx context.Context, path string) error {
	return m.DeleteFile(ctx, path)
}

// GetSnapshotURL implements SnapshotStorage interface
func (m *MockRustFSClient) GetSnapshotURL(path string) string {
	return m.GetFileURL(path)
}

// Close closes the mock client
func (m *MockRustFSClient) Close() error {
	m.Reset()
	return nil
}

// CheckHealth performs health check on mock storage
func (m *MockRustFSClient) CheckHealth(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return m.failError
	}

	return nil
}

// ListFiles lists files in mock storage (additional method for testing)
func (m *MockRustFSClient) ListFiles(ctx context.Context, prefix string, limit int) ([]*types.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return nil, m.failError
	}

	var files []*types.FileInfo
	count := 0

	for path, fileInfo := range m.files {
		if prefix != "" && !containsPath(path, prefix) {
			continue
		}
		if limit > 0 && count >= limit {
			break
		}
		files = append(files, fileInfo)
		count++
	}

	return files, nil
}

// CopyFile copies a file within mock storage (additional method for testing)
func (m *MockRustFSClient) CopyFile(ctx context.Context, sourcePath, destPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		m.shouldFail = false // Reset failure mode
		return m.failError
	}

	sourceFile, exists := m.files[sourcePath]
	if !exists {
		return fmt.Errorf("source file not found: %s", sourcePath)
	}

	// Create copy
	destFile := &types.FileInfo{
		Path:         destPath,
		Size:         sourceFile.Size,
		ContentType:  sourceFile.ContentType,
		ETag:         fmt.Sprintf("etag-%d", time.Now().UnixNano()),
		LastModified: time.Now(),
		Metadata:     sourceFile.Metadata,
	}

	m.files[destPath] = destFile
	return nil
}

// containsPath checks if path contains prefix
func containsPath(path, prefix string) bool {
	if len(prefix) > len(path) {
		return false
	}
	return path[:len(prefix)] == prefix
}

// MockRustFSClientBuilder helps build mock clients with predefined data
type MockRustFSClientBuilder struct {
	client *MockRustFSClient
}

// NewMockRustFSClientBuilder creates a new mock client builder
func NewMockRustFSClientBuilder() *MockRustFSClientBuilder {
	return &MockRustFSClientBuilder{
		client: NewMockRustFSClient(),
	}
}

// WithFile adds a predefined file to the mock client
func (b *MockRustFSClientBuilder) WithFile(path string, size int64, contentType string) *MockRustFSClientBuilder {
	fileInfo := &types.FileInfo{
		Path:         path,
		Size:         size,
		ContentType:  contentType,
		ETag:         fmt.Sprintf("etag-%d", time.Now().UnixNano()),
		LastModified: time.Now(),
		Metadata:     make(map[string]interface{}),
	}
	b.client.files[path] = fileInfo
	return b
}

// WithFailure sets the mock client to fail
func (b *MockRustFSClientBuilder) WithFailure(err error) *MockRustFSClientBuilder {
	b.client.shouldFail = true
	b.client.failError = err
	return b
}

// Build creates the mock client
func (b *MockRustFSClientBuilder) Build() *MockRustFSClient {
	return b.client
}
