package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/garyjdn/go-apperror"
	"github.com/garyjdn/go-rustfs/config"
	"github.com/garyjdn/go-rustfs/types"
	"github.com/garyjdn/go-rustfs/utils"
)

// RustFSClient implements the FileStorage interface using AWS SDK for Go v2
type RustFSClient struct {
	client *s3.Client
	config *config.RustFSConfig
}

// NewRustFSClient creates a new RustFS client
func NewRustFSClient(cfg *config.RustFSConfig) *RustFSClient {
	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		// This should theoretically not happen with static credentials
		panic(fmt.Sprintf("failed to load AWS config: %v", err))
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.BaseURL)
		o.UsePathStyle = true // Required for MinIO/RustFS
	})

	return &RustFSClient{
		client: client,
		config: cfg,
	}
}

// UploadFile uploads a file to RustFS
func (c *RustFSClient) UploadFile(ctx context.Context, req *types.UploadRequest) (*types.UploadResponse, error) {
	// Create a buffer to store the file content if it's not already a Seekable reader
	// S3 PutObject requires a ReadSeeker for optimal performance and to calculate content length automatically
	var body io.Reader = req.File
	var size int64 = req.FileSize

	// If size is unknown (0) or we need to ensure we can read it, read into buffer
	// optimized: if it's already bytes.Reader, we can use it directly, but here we keep it safe
	if size == 0 {
		buf := new(bytes.Buffer)
		n, err := io.Copy(buf, req.File)
		if err != nil {
			return nil, apperror.NewAppError(500, "FILE_READ_ERROR", err)
		}
		body = bytes.NewReader(buf.Bytes())
		size = n
	}

	contentType := "application/octet-stream"
	if req.ContentType != "" {
		contentType = req.ContentType
	}

	// Prepare metadata
	metadata := make(map[string]string)
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	// Create PutObject input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.config.BucketName),
		Key:         aws.String(req.BucketPath),
		Body:        body,
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	}

	// Upload to S3
	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return nil, apperror.NewAppError(500, "UPLOAD_FAILED", err)
	}

	// Return response
	return &types.UploadResponse{
		Path:         req.BucketPath,
		URL:          c.GetFileURL(req.BucketPath),
		Size:         size,
		ContentType:  contentType,
		ETag:         "", // AWS SDK doesn't always return ETag in output struct easily without pointers
		LastModified: time.Now(),
		Metadata:     req.Metadata,
	}, nil
}

// DeleteFile deletes a file from RustFS
func (c *RustFSClient) DeleteFile(ctx context.Context, path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.config.BucketName),
		Key:    aws.String(path),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return apperror.NewAppError(500, "DELETE_FAILED", err)
	}

	return nil
}

// GetFileURL returns the public URL for a file
func (c *RustFSClient) GetFileURL(path string) string {
	// Construct URL manually as S3 doesn't return it directly
	// Format: BaseURL/BucketName/Path
	baseURL := c.config.BaseURL
	// Remove trailing slash if present
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	return fmt.Sprintf("%s/%s/%s", baseURL, c.config.BucketName, path)
}

// GetFileInfo retrieves file information from RustFS
func (c *RustFSClient) GetFileInfo(ctx context.Context, path string) (*types.FileInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.config.BucketName),
		Key:    aws.String(path),
	}

	output, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, apperror.NewAppError(404, "FILE_NOT_FOUND", err)
	}

	metadata := make(map[string]interface{})
	for k, v := range output.Metadata {
		metadata[k] = v
	}

	return &types.FileInfo{
		Path:         path,
		Size:         *output.ContentLength,
		ContentType:  *output.ContentType,
		ETag:         *output.ETag,
		LastModified: *output.LastModified,
		Metadata:     metadata,
	}, nil
}

// UploadSnapshot uploads a snapshot (specific implementation for interface compliance)
func (c *RustFSClient) UploadSnapshot(ctx context.Context, file io.Reader, filename string) (string, error) {
	// Generate path
	path := utils.GenerateFilePath(filename, "snapshots")

	// Calculate size if possible, otherwise read all
	var size int64
	var body io.Reader

	if seeker, ok := file.(io.Seeker); ok {
		// It's a seeker, try to get size
		current, _ := seeker.Seek(0, io.SeekCurrent)
		end, _ := seeker.Seek(0, io.SeekEnd)
		size = end
		seeker.Seek(current, io.SeekStart)
		body = file
	} else {
		// Read into buffer
		buf := new(bytes.Buffer)
		n, err := io.Copy(buf, file)
		if err != nil {
			return "", err
		}
		size = n
		body = bytes.NewReader(buf.Bytes())
	}

	req := &types.UploadRequest{
		File:        body,
		Filename:    filename,
		ContentType: "image/jpeg", // Default for snapshots
		FileSize:    size,
		BucketPath:  path,
	}

	resp, err := c.UploadFile(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Path, nil
}

// DeleteSnapshot deletes a snapshot
func (c *RustFSClient) DeleteSnapshot(ctx context.Context, path string) error {
	return c.DeleteFile(ctx, path)
}

// GetSnapshotURL returns the public URL for a snapshot
func (c *RustFSClient) GetSnapshotURL(path string) string {
	return c.GetFileURL(path)
}

// CheckHealth checks if the storage service is available
func (c *RustFSClient) CheckHealth(ctx context.Context) error {
	// Check if bucket exists/is accessible
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.config.BucketName),
	})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}
