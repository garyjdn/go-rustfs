package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/garyjdn/go-apperror"
	"github.com/garyjdn/go-rustfs/config"
	"github.com/garyjdn/go-rustfs/types"
	"github.com/garyjdn/go-rustfs/utils"
)

// RustFSClient is the concrete implementation of FileStorage interface
type RustFSClient struct {
	config     *config.RustFSConfig
	httpClient *http.Client
}

// NewRustFSClient creates a new RustFS client instance
func NewRustFSClient(config *config.RustFSConfig) *RustFSClient {
	return &RustFSClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// UploadFile uploads a file to RustFS storage
func (c *RustFSClient) UploadFile(ctx context.Context, req *types.UploadRequest) (*types.UploadResponse, error) {
	// Create a buffer to store the file content
	var buf bytes.Buffer

	// Copy file content to buffer
	if _, err := io.Copy(&buf, req.File); err != nil {
		return nil, apperror.NewAppError(500, "FILE_READ_ERROR", err)
	}

	// Prepare upload request
	uploadURL := fmt.Sprintf("%s/api/v1/buckets/%s/upload", c.config.BaseURL, c.config.BucketName)

	// Create multipart form
	body := &bytes.Buffer{}
	contentType := "application/octet-stream"

	// If content type is not provided, try to detect it
	if req.ContentType != "" {
		contentType = req.ContentType
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, apperror.NewAppError(500, "REQUEST_CREATION_ERROR", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("X-Filename", req.Filename)
	httpReq.Header.Set("X-File-Path", req.BucketPath)
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Add metadata headers
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, apperror.NewAppError(500, "METADATA_ENCODE_ERROR", err)
		}
		httpReq.Header.Set("X-Metadata", string(metadataJSON))
	}

	// Set request body
	httpReq.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	httpReq.ContentLength = int64(buf.Len())

	// Execute request with retry
	var resp *http.Response
	result := utils.RetryWithBackoffWithContext(ctx, func(ctx context.Context) error {
		var retryErr error
		resp, retryErr = c.httpClient.Do(httpReq)
		if retryErr != nil {
			return retryErr
		}
		return nil
	}, c.config.RetryConfig)

	if !result.Success {
		return nil, apperror.NewAppError(500, "UPLOAD_REQUEST_FAILED", result.LastError)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, apperror.NewAppError(resp.StatusCode, "UPLOAD_FAILED",
			fmt.Errorf("RustFS API error: %s", string(bodyBytes)))
	}

	// Parse response
	var uploadResp types.UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, apperror.NewAppError(500, "RESPONSE_PARSE_ERROR", err)
	}

	// Set size in response
	uploadResp.Size = req.FileSize

	return &uploadResp, nil
}

// DeleteFile deletes a file from RustFS storage
func (c *RustFSClient) DeleteFile(ctx context.Context, path string) error {
	deleteURL := fmt.Sprintf("%s/api/v1/files/%s", c.config.BaseURL, path)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return apperror.NewAppError(500, "REQUEST_CREATION_ERROR", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request with retry
	var resp *http.Response
	result := utils.RetryWithBackoffWithContext(ctx, func(ctx context.Context) error {
		var retryErr error
		resp, retryErr = c.httpClient.Do(httpReq)
		if retryErr != nil {
			return retryErr
		}
		return nil
	}, c.config.RetryConfig)

	if !result.Success {
		return apperror.NewAppError(500, "DELETE_REQUEST_FAILED", result.LastError)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return apperror.NewAppError(resp.StatusCode, "DELETE_FAILED",
			fmt.Errorf("RustFS API error: %s", string(bodyBytes)))
	}

	return nil
}

// GetFileURL returns the public URL for a file
func (c *RustFSClient) GetFileURL(path string) string {
	return fmt.Sprintf("%s/%s/%s", c.config.BaseURL, c.config.BucketName, path)
}

// GetFileInfo retrieves file information from RustFS storage
func (c *RustFSClient) GetFileInfo(ctx context.Context, path string) (*types.FileInfo, error) {
	infoURL := fmt.Sprintf("%s/api/v1/files/%s/info", c.config.BaseURL, path)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
	if err != nil {
		return nil, apperror.NewAppError(500, "REQUEST_CREATION_ERROR", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request with retry
	var resp *http.Response
	result := utils.RetryWithBackoffWithContext(ctx, func(ctx context.Context) error {
		var retryErr error
		resp, retryErr = c.httpClient.Do(httpReq)
		if retryErr != nil {
			return retryErr
		}
		return nil
	}, c.config.RetryConfig)

	if !result.Success {
		return nil, apperror.NewAppError(500, "INFO_REQUEST_FAILED", result.LastError)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, apperror.NewAppError(resp.StatusCode, "GET_INFO_FAILED",
			fmt.Errorf("RustFS API error: %s", string(bodyBytes)))
	}

	// Parse response
	var fileInfo types.FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
		return nil, apperror.NewAppError(500, "RESPONSE_PARSE_ERROR", err)
	}

	return &fileInfo, nil
}

// CheckHealth performs a health check on the RustFS service
func (c *RustFSClient) CheckHealth(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/health", c.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return apperror.NewAppError(500, "HEALTH_CHECK_REQUEST_ERROR", err)
	}

	// Execute request with shorter timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return apperror.NewAppError(500, "HEALTH_CHECK_FAILED", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperror.NewAppError(resp.StatusCode, "HEALTH_CHECK_UNHEALTHY",
			fmt.Errorf("RustFS service returned status: %d", resp.StatusCode))
	}

	return nil
}

// Close closes the client and performs cleanup
func (c *RustFSClient) Close() error {
	// No specific cleanup needed for HTTP client
	return nil
}

// GetConfig returns the client configuration
func (c *RustFSClient) GetConfig() *config.RustFSConfig {
	return c.config
}

// ListFiles lists files in a directory (optional implementation)
func (c *RustFSClient) ListFiles(ctx context.Context, prefix string, limit int) ([]*types.FileInfo, error) {
	listURL := fmt.Sprintf("%s/api/v1/buckets/%s/files", c.config.BaseURL, c.config.BucketName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, apperror.NewAppError(500, "REQUEST_CREATION_ERROR", err)
	}

	// Add query parameters
	q := httpReq.URL.Query()
	if prefix != "" {
		q.Add("prefix", prefix)
	}
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	httpReq.URL.RawQuery = q.Encode()

	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request with retry
	var resp *http.Response
	result := utils.RetryWithBackoffWithContext(ctx, func(ctx context.Context) error {
		var retryErr error
		resp, retryErr = c.httpClient.Do(httpReq)
		if retryErr != nil {
			return retryErr
		}
		return nil
	}, c.config.RetryConfig)

	if !result.Success {
		return nil, apperror.NewAppError(500, "LIST_REQUEST_FAILED", result.LastError)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, apperror.NewAppError(resp.StatusCode, "LIST_FAILED",
			fmt.Errorf("RustFS API error: %s", string(bodyBytes)))
	}

	// Parse response
	var listResponse struct {
		Files []*types.FileInfo `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&listResponse); err != nil {
		return nil, apperror.NewAppError(500, "RESPONSE_PARSE_ERROR", err)
	}

	return listResponse.Files, nil
}

// CopyFile copies a file within RustFS storage (optional implementation)
func (c *RustFSClient) CopyFile(ctx context.Context, sourcePath, destPath string) error {
	copyURL := fmt.Sprintf("%s/api/v1/files/copy", c.config.BaseURL)

	copyReq := struct {
		Source string `json:"source"`
		Dest   string `json:"dest"`
	}{
		Source: sourcePath,
		Dest:   destPath,
	}

	reqBody, err := json.Marshal(copyReq)
	if err != nil {
		return apperror.NewAppError(500, "REQUEST_ENCODE_ERROR", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", copyURL, bytes.NewReader(reqBody))
	if err != nil {
		return apperror.NewAppError(500, "REQUEST_CREATION_ERROR", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request with retry
	var resp *http.Response
	result := utils.RetryWithBackoffWithContext(ctx, func(ctx context.Context) error {
		var retryErr error
		resp, retryErr = c.httpClient.Do(httpReq)
		if retryErr != nil {
			return retryErr
		}
		return nil
	}, c.config.RetryConfig)

	if !result.Success {
		return apperror.NewAppError(500, "COPY_REQUEST_FAILED", result.LastError)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return apperror.NewAppError(resp.StatusCode, "COPY_FAILED",
			fmt.Errorf("RustFS API error: %s", string(bodyBytes)))
	}

	return nil
}
