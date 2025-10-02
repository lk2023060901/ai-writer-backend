package minio

import (
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// bucketNameRegex validates bucket names according to AWS S3 rules
	bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,61}[a-z0-9]$`)

	// invalidBucketNamePrefixes are invalid bucket name prefixes
	invalidBucketNamePrefixes = []string{"xn--", "sthree-", "sthree-configurator"}

	// invalidBucketNameSuffixes are invalid bucket name suffixes
	invalidBucketNameSuffixes = []string{"-s3alias", "--ol-s3"}
)

// ValidateBucketName validates a bucket name according to AWS S3 naming rules
func ValidateBucketName(bucketName string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	// Length check
	if len(bucketName) < 3 || len(bucketName) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters long")
	}

	// Regex pattern check
	if !bucketNameRegex.MatchString(bucketName) {
		return fmt.Errorf("bucket name must start and end with a lowercase letter or number, and can only contain lowercase letters, numbers, and hyphens")
	}

	// Check for invalid prefixes
	for _, prefix := range invalidBucketNamePrefixes {
		if strings.HasPrefix(bucketName, prefix) {
			return fmt.Errorf("bucket name cannot start with '%s'", prefix)
		}
	}

	// Check for invalid suffixes
	for _, suffix := range invalidBucketNameSuffixes {
		if strings.HasSuffix(bucketName, suffix) {
			return fmt.Errorf("bucket name cannot end with '%s'", suffix)
		}
	}

	// Check for consecutive hyphens
	if strings.Contains(bucketName, "--") {
		return fmt.Errorf("bucket name cannot contain consecutive hyphens")
	}

	// Check for IP address format
	if isIPAddress(bucketName) {
		return fmt.Errorf("bucket name cannot be formatted as an IP address")
	}

	return nil
}

// ValidateObjectName validates an object name
func ValidateObjectName(objectName string) error {
	if objectName == "" {
		return fmt.Errorf("object name cannot be empty")
	}

	// Length check (S3 allows up to 1024 characters)
	if len(objectName) > 1024 {
		return fmt.Errorf("object name cannot exceed 1024 characters")
	}

	// Check for invalid characters (null bytes)
	if strings.Contains(objectName, "\x00") {
		return fmt.Errorf("object name cannot contain null bytes")
	}

	return nil
}

// isIPAddress checks if a string is formatted as an IP address
func isIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}

		// Check if all characters are digits
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}

		// Check range (0-255)
		var num int
		fmt.Sscanf(part, "%d", &num)
		if num < 0 || num > 255 {
			return false
		}
	}

	return true
}

// DetectContentType detects the content type of a file based on its extension
func DetectContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return "application/octet-stream"
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

// ProgressFunc is a callback function for tracking upload/download progress
type ProgressFunc func(current, total int64)

// ProgressReader wraps an io.Reader and reports progress through a callback
type ProgressReader struct {
	reader   io.Reader
	size     int64
	current  int64
	callback ProgressFunc
}

// NewProgressReader creates a new ProgressReader
func NewProgressReader(reader io.Reader, size int64, callback ProgressFunc) io.Reader {
	if callback == nil {
		return reader
	}

	return &ProgressReader{
		reader:   reader,
		size:     size,
		current:  0,
		callback: callback,
	}
}

// Read implements io.Reader
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	if pr.callback != nil {
		pr.callback(pr.current, pr.size)
	}

	return n, err
}

// FormatBytes formats bytes to human-readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SanitizeObjectName sanitizes an object name by removing invalid characters
func SanitizeObjectName(objectName string) string {
	// Replace null bytes
	objectName = strings.ReplaceAll(objectName, "\x00", "")

	// Trim leading and trailing slashes
	objectName = strings.Trim(objectName, "/")

	// Replace multiple consecutive slashes with a single slash
	for strings.Contains(objectName, "//") {
		objectName = strings.ReplaceAll(objectName, "//", "/")
	}

	return objectName
}

// GenerateObjectKey generates an object key from a file path
func GenerateObjectKey(filePath, prefix string) string {
	filename := filepath.Base(filePath)
	if prefix == "" {
		return filename
	}

	prefix = strings.TrimSuffix(prefix, "/")
	return fmt.Sprintf("%s/%s", prefix, filename)
}
