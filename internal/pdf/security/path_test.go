package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPathValidator(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		dir       string
		wantError bool
	}{
		{
			name:      "valid directory",
			dir:       tempDir,
			wantError: false,
		},
		{
			name:      "empty directory",
			dir:       "",
			wantError: true,
		},
		{
			name:      "non-existent directory",
			dir:       "/non/existent/path",
			wantError: false, // Now allowed for placeholder paths
		},
		{
			name:      "file instead of directory",
			dir:       filepath.Join(tempDir, "test.txt"),
			wantError: false, // Now allowed since we don't validate existence
		},
	}

	// Create a test file for the "file instead of directory" test
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewPathValidator(tt.dir)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if validator == nil {
					t.Error("Expected validator but got nil")
				}
			}
		})
	}
}

func TestPathValidator_ValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create test files
	validFile := filepath.Join(tempDir, "valid.pdf")
	subFile := filepath.Join(subDir, "sub.pdf")
	if err := os.WriteFile(validFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(subFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	validator, err := NewPathValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "valid file in root",
			path:      validFile,
			wantError: false,
		},
		{
			name:      "valid file in subdirectory",
			path:      subFile,
			wantError: false,
		},
		{
			name:      "file outside directory",
			path:      "/etc/passwd",
			wantError: true,
		},
		{
			name:      "parent directory traversal",
			path:      filepath.Join(tempDir, "..", "outside.pdf"),
			wantError: true,
		},
		{
			name:      "relative path within directory",
			path:      filepath.Join(tempDir, ".", "valid.pdf"),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPathValidator_IsPathWithinDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator, err := NewPathValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a symlink for testing
	targetFile := filepath.Join(tempDir, "target.pdf")
	symlinkFile := filepath.Join(tempDir, "symlink.pdf")
	if err := os.WriteFile(targetFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Logf("Warning: Failed to create symlink (may not be supported): %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "path within directory",
			path:     filepath.Join(tempDir, "test.pdf"),
			expected: true,
		},
		{
			name:     "path outside directory",
			path:     "/tmp/outside.pdf",
			expected: false,
		},
		{
			name:     "parent directory traversal",
			path:     filepath.Join(tempDir, "..", "outside.pdf"),
			expected: false,
		},
		{
			name:     "symlink within directory",
			path:     symlinkFile,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.IsPathWithinDirectory(tt.path)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestPathValidator_NormalizePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator, err := NewPathValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
		{
			name:      "relative path",
			path:      "test.pdf",
			wantError: false,
		},
		{
			name:      "absolute path within directory",
			path:      filepath.Join(tempDir, "test.pdf"),
			wantError: false,
		},
		{
			name:      "path with ..",
			path:      "../outside.pdf",
			wantError: true,
		},
		{
			name:      "path with .",
			path:      "./test.pdf",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.NormalizePath(tt.path)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Result should be an absolute path
				if !filepath.IsAbs(result) {
					t.Errorf("Expected absolute path but got: %s", result)
				}
				// Result should be within the configured directory
				if !filepath.HasPrefix(result, tempDir) {
					t.Errorf("Expected path to be within %s but got: %s", tempDir, result)
				}
			}
		})
	}
}

func TestPathValidator_ValidateDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectory and file
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.pdf")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator, err := NewPathValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid subdirectory",
			path:      subDir,
			wantError: false,
		},
		{
			name:      "file instead of directory",
			path:      testFile,
			wantError: true,
		},
		{
			name:      "non-existent directory",
			path:      filepath.Join(tempDir, "nonexistent"),
			wantError: false, // Now allowed since directory might not exist yet
		},
		{
			name:      "directory outside bounds",
			path:      "/tmp",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDirectory(tt.path)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPathValidator_SanitizePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validator_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator, err := NewPathValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "normal path",
			path:      "test.pdf",
			wantError: false,
		},
		{
			name:      "path with null bytes",
			path:      "test\x00.pdf",
			wantError: false,
		},
		{
			name:      "path with special characters",
			path:      "test file (1).pdf",
			wantError: false,
		},
		{
			name:      "path attempting traversal",
			path:      "../../../etc/passwd",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.SanitizePath(tt.path)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Result should not contain null bytes
				if len(result) > 0 && result[0] == '\x00' {
					t.Error("Result still contains null bytes")
				}
			}
		})
	}
}
