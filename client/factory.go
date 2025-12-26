package client

import (
	"context"
	"fmt"

	"github.com/garyjdn/go-rustfs/audit"
	"github.com/garyjdn/go-rustfs/config"
	"github.com/garyjdn/go-rustfs/types"
)

// ClientFactory creates different types of RustFS clients
type ClientFactory struct{}

// NewClientFactory creates a new client factory
func NewClientFactory() *ClientFactory {
	return &ClientFactory{}
}

// CreateProductionClient creates a production RustFS client with audit logging
func (f *ClientFactory) CreateProductionClient(serviceName string) (*AuditableRustFSClient, error) {
	// Load configuration
	cfg := config.GetConfigForService(serviceName)

	// Create base client
	baseClient := NewRustFSClient(cfg)

	// Create audit logger if enabled
	var auditLogger *audit.RustFSAuditLogger
	if cfg.EnableAudit {
		// For now, we'll create a simple audit logger
		// In production, this should be replaced with actual audit logger implementation
		auditLogger = audit.NewRustFSAuditLogger(cfg.AuditService, nil, cfg.AuditMetadata)
	}

	// Create auditable client
	return NewAuditableRustFSClient(baseClient, auditLogger, cfg, serviceName), nil
}

// CreateDevelopmentClient creates a development RustFS client (mock)
func (f *ClientFactory) CreateDevelopmentClient(serviceName string) (*AuditableRustFSClient, error) {
	// Load configuration
	cfg := config.GetConfigForService(serviceName)

	// Create mock client
	mockClient := NewMockRustFSClient()

	// Create audit logger for development (console only)
	var auditLogger *audit.RustFSAuditLogger
	if cfg.EnableAudit {
		auditLogger = audit.NewRustFSAuditLogger(cfg.AuditService, nil, cfg.AuditMetadata)
	}

	// Create auditable client
	return NewAuditableRustFSClient(mockClient, auditLogger, cfg, serviceName), nil
}

// CreateTestClient creates a test RustFS client (mock with predefined data)
func (f *ClientFactory) CreateTestClient(serviceName string, testData *TestData) (*AuditableRustFSClient, error) {
	// Load configuration
	cfg := config.GetConfigForService(serviceName)

	// Create mock client with test data
	mockClient := NewMockRustFSClient()

	// Add test files if provided
	if testData != nil {
		for _, file := range testData.Files {
			mockClient.files[file.Path] = file
		}
	}

	// Create audit logger for testing (console only, minimal)
	var auditLogger *audit.RustFSAuditLogger
	if cfg.EnableAudit {
		auditLogger = audit.NewRustFSAuditLogger(cfg.AuditService, nil, cfg.AuditMetadata)
	}

	// Create auditable client
	return NewAuditableRustFSClient(mockClient, auditLogger, cfg, serviceName), nil
}

// CreateClientFromConfig creates a client from custom configuration
func (f *ClientFactory) CreateClientFromConfig(cfg *config.RustFSConfig, serviceName string, useMock bool) (*AuditableRustFSClient, error) {
	var baseClient FileStorage

	if useMock {
		baseClient = NewMockRustFSClient()
	} else {
		baseClient = NewRustFSClient(cfg)
	}

	// Create audit logger if enabled
	var auditLogger *audit.RustFSAuditLogger
	if cfg.EnableAudit {
		auditLogger = audit.NewRustFSAuditLogger(cfg.AuditService, nil, cfg.AuditMetadata)
	}

	// Create auditable client
	return NewAuditableRustFSClient(baseClient, auditLogger, cfg, serviceName), nil
}

// TestData represents test data for mock client
type TestData struct {
	Files []*types.FileInfo
}

// NewTestData creates new test data
func NewTestData() *TestData {
	return &TestData{
		Files: make([]*types.FileInfo, 0),
	}
}

// AddFile adds a file to test data
func (td *TestData) AddFile(path string, size int64, contentType string) *TestData {
	file := &types.FileInfo{
		Path:        path,
		Size:        size,
		ContentType: contentType,
		ETag:        fmt.Sprintf("test-etag-%d", len(td.Files)),
		Metadata:    make(map[string]interface{}),
	}
	td.Files = append(td.Files, file)
	return td
}

// ClientType represents different client types
type ClientType string

const (
	ClientTypeProduction  ClientType = "production"
	ClientTypeDevelopment ClientType = "development"
	ClientTypeTest        ClientType = "test"
)

// CreateClient creates a client based on type
func (f *ClientFactory) CreateClient(clientType ClientType, serviceName string, testData *TestData) (*AuditableRustFSClient, error) {
	switch clientType {
	case ClientTypeProduction:
		return f.CreateProductionClient(serviceName)
	case ClientTypeDevelopment:
		return f.CreateDevelopmentClient(serviceName)
	case ClientTypeTest:
		return f.CreateTestClient(serviceName, testData)
	default:
		return nil, fmt.Errorf("unsupported client type: %s", clientType)
	}
}

// GetClientTypeFromEnvironment determines client type from environment
func GetClientTypeFromEnvironment() ClientType {
	envType := "development" // Default

	// Check environment variable
	if value := ""; value != "" {
		envType = value
	}

	switch envType {
	case "production", "prod":
		return ClientTypeProduction
	case "test":
		return ClientTypeTest
	default:
		return ClientTypeDevelopment
	}
}

// CreateClientFromEnvironment creates a client based on environment
func (f *ClientFactory) CreateClientFromEnvironment(serviceName string, testData *TestData) (*AuditableRustFSClient, error) {
	clientType := GetClientTypeFromEnvironment()
	return f.CreateClient(clientType, serviceName, testData)
}

// CheckClientHealth performs health check on a client
func CheckClientHealth(ctx context.Context, client interface{}) error {
	// Try to get underlying client if it's an auditable client
	if auditableClient, ok := client.(*AuditableRustFSClient); ok {
		return auditableClient.HealthCheck(ctx)
	}

	// Try direct interface
	if healthChecker, ok := client.(interface {
		CheckHealth(ctx context.Context) error
	}); ok {
		return healthChecker.CheckHealth(ctx)
	}

	return fmt.Errorf("client does not support health checking")
}
