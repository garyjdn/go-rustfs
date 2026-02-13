package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// RustFSConfig represents configuration for RustFS client
type RustFSConfig struct {
	// Connection settings
	BaseURL    string `json:"base_url" env:"RUSTFS_BASE_URL"`
	AccessKey  string `json:"access_key" env:"RUSTFS_ACCESS_KEY"`
	SecretKey  string `json:"secret_key" env:"RUSTFS_SECRET_KEY"`
	Region     string `json:"region" env:"RUSTFS_REGION"`
	BucketName string `json:"bucket_name" env:"RUSTFS_BUCKET_NAME"`

	// Performance settings
	Timeout    time.Duration `json:"timeout" env:"RUSTFS_TIMEOUT"`
	RetryCount int           `json:"retry_count" env:"RUSTFS_RETRY_COUNT"`

	// File validation settings
	MaxFileSize    int64    `json:"max_file_size" env:"RUSTFS_MAX_FILE_SIZE"`
	AllowedTypes   []string `json:"allowed_types" env:"RUSTFS_ALLOWED_TYPES"`
	ScanForMalware bool     `json:"scan_for_malware" env:"RUSTFS_SCAN_MALWARE"`

	// Audit settings
	EnableAudit   bool                   `json:"enable_audit" env:"RUSTFS_ENABLE_AUDIT"`
	AuditService  string                 `json:"audit_service" env:"RUSTFS_AUDIT_SERVICE"`
	AuditMetadata map[string]interface{} `json:"audit_metadata"`

	// Security settings
	EnableEncryption bool     `json:"enable_encryption" env:"RUSTFS_ENABLE_ENCRYPTION"`
	EncryptionKey    string   `json:"encryption_key" env:"RUSTFS_ENCRYPTION_KEY"`
	AllowedOrigins   []string `json:"allowed_origins" env:"RUSTFS_ALLOWED_ORIGINS"`

	// Performance tuning
	ConcurrentUploads int           `json:"concurrent_uploads" env:"RUSTFS_CONCURRENT_UPLOADS"`
	ChunkSize         int           `json:"chunk_size" env:"RUSTFS_CHUNK_SIZE"`
	CompressionLevel  int           `json:"compression_level" env:"RUSTFS_COMPRESSION_LEVEL"`
	CacheEnabled      bool          `json:"cache_enabled" env:"RUSTFS_CACHE_ENABLED"`
	CacheTTL          time.Duration `json:"cache_ttl" env:"RUSTFS_CACHE_TTL"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *RustFSConfig {
	config := &RustFSConfig{
		// Connection defaults
		BaseURL:    getEnvOrDefault("RUSTFS_BASE_URL", "http://localhost:8080"),
		AccessKey:  getEnvOrDefault("RUSTFS_ACCESS_KEY", ""),
		SecretKey:  getEnvOrDefault("RUSTFS_SECRET_KEY", ""),
		Region:     getEnvOrDefault("RUSTFS_REGION", "us-east-1"),
		BucketName: getEnvOrDefault("RUSTFS_BUCKET_NAME", "default"),

		// Performance defaults
		Timeout:    getDurationEnvOrDefault("RUSTFS_TIMEOUT", 30*time.Second),
		RetryCount: getIntEnvOrDefault("RUSTFS_RETRY_COUNT", 3),

		// File validation defaults
		MaxFileSize:    getInt64EnvOrDefault("RUSTFS_MAX_FILE_SIZE", 100*1024*1024), // 100MB
		AllowedTypes:   getStringSliceEnvOrDefault("RUSTFS_ALLOWED_TYPES", []string{"image/*"}),
		ScanForMalware: getBoolEnvOrDefault("RUSTFS_SCAN_MALWARE", false),

		// Audit defaults
		EnableAudit:  getBoolEnvOrDefault("RUSTFS_ENABLE_AUDIT", true),
		AuditService: getEnvOrDefault("RUSTFS_AUDIT_SERVICE", "rustfs-client"),
		AuditMetadata: map[string]interface{}{
			"version":     "1.0.0",
			"environment": getEnvOrDefault("ENVIRONMENT", "development"),
		},

		// Security defaults
		EnableEncryption: getBoolEnvOrDefault("RUSTFS_ENABLE_ENCRYPTION", false),
		EncryptionKey:    getEnvOrDefault("RUSTFS_ENCRYPTION_KEY", ""),
		AllowedOrigins:   getStringSliceEnvOrDefault("RUSTFS_ALLOWED_ORIGINS", []string{"*"}),

		// Performance tuning defaults
		ConcurrentUploads: getIntEnvOrDefault("RUSTFS_CONCURRENT_UPLOADS", 5),
		ChunkSize:         getIntEnvOrDefault("RUSTFS_CHUNK_SIZE", 1024*1024), // 1MB
		CompressionLevel:  getIntEnvOrDefault("RUSTFS_COMPRESSION_LEVEL", 6),
		CacheEnabled:      getBoolEnvOrDefault("RUSTFS_CACHE_ENABLED", true),
		CacheTTL:          getDurationEnvOrDefault("RUSTFS_CACHE_TTL", 1*time.Hour),
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid RustFS configuration: %v", err))
	}

	return config
}

// Validate validates the configuration
func (c *RustFSConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("RUSTFS_BASE_URL is required")
	}

	if c.AccessKey == "" {
		return fmt.Errorf("RUSTFS_ACCESS_KEY is required")
	}

	if c.SecretKey == "" {
		return fmt.Errorf("RUSTFS_SECRET_KEY is required")
	}

	if c.BucketName == "" {
		return fmt.Errorf("RUSTFS_BUCKET_NAME is required")
	}

	if c.MaxFileSize <= 0 {
		return fmt.Errorf("RUSTFS_MAX_FILE_SIZE must be positive")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("RUSTFS_TIMEOUT must be positive")
	}

	if c.RetryCount < 0 {
		return fmt.Errorf("RUSTFS_RETRY_COUNT cannot be negative")
	}

	if c.EnableEncryption && c.EncryptionKey == "" {
		return fmt.Errorf("RUSTFS_ENCRYPTION_KEY is required when encryption is enabled")
	}

	if c.ConcurrentUploads <= 0 {
		return fmt.Errorf("RUSTFS_CONCURRENT_UPLOADS must be positive")
	}

	if c.ChunkSize <= 0 {
		return fmt.Errorf("RUSTFS_CHUNK_SIZE must be positive")
	}

	if c.CompressionLevel < 0 || c.CompressionLevel > 9 {
		return fmt.Errorf("RUSTFS_COMPRESSION_LEVEL must be between 0 and 9")
	}

	return nil
}

// IsAllowedType checks if the content type is allowed
func (c *RustFSConfig) IsAllowedType(contentType string) bool {
	for _, allowedType := range c.AllowedTypes {
		if matchContentType(allowedType, contentType) {
			return true
		}
	}
	return false
}

// IsAllowedOrigin checks if the origin is allowed
func (c *RustFSConfig) IsAllowedOrigin(origin string) bool {
	for _, allowedOrigin := range c.AllowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64EnvOrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceEnvOrDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

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

// GetConfigForService returns service-specific configuration
func GetConfigForService(serviceName string) *RustFSConfig {
	config := LoadConfig()

	// Override service-specific settings
	if servicePrefix := os.Getenv("RUSTFS_SERVICE_PREFIX"); servicePrefix != "" {
		config.BucketName = servicePrefix + "-" + config.BucketName
	}

	// Add service-specific audit metadata
	if config.AuditMetadata == nil {
		config.AuditMetadata = make(map[string]interface{})
	}
	config.AuditMetadata["service_name"] = serviceName

	return config
}
