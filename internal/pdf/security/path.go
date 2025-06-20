package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathValidator provides security validation for file paths
type PathValidator struct {
	configuredDirectory string
}

// NewPathValidator creates a new path validator for the given directory
func NewPathValidator(configuredDirectory string) (*PathValidator, error) {
	if configuredDirectory == "" {
		return nil, fmt.Errorf("configured directory cannot be empty")
	}

	// Use the directory as provided - don't require it to exist
	// This allows for placeholders and directories that may be created later
	return &PathValidator{
		configuredDirectory: configuredDirectory,
	}, nil
}

// ValidatePath checks if a path is within the configured directory
func (v *PathValidator) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// If configured directory doesn't exist yet, skip validation
	if _, err := os.Stat(v.configuredDirectory); os.IsNotExist(err) {
		return nil
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path is within configured directory
	isWithin, err := v.IsPathWithinDirectory(absPath)
	if err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	if !isWithin {
		return fmt.Errorf("path is outside configured directory: %s", path)
	}

	return nil
}

// IsPathWithinDirectory checks if a path is within the configured directory
func (v *PathValidator) IsPathWithinDirectory(path string) (bool, error) {
	// If configured directory doesn't exist yet, allow any path
	if _, err := os.Stat(v.configuredDirectory); os.IsNotExist(err) {
		return true, nil
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Resolve configured directory to absolute path
	absConfigDir, err := filepath.Abs(v.configuredDirectory)
	if err != nil {
		return false, fmt.Errorf("failed to resolve configured directory: %w", err)
	}

	// Clean paths to remove any .. or . segments
	cleanPath := filepath.Clean(absPath)
	cleanDir := filepath.Clean(absConfigDir)

	// Handle symlinks - evaluate the real paths for both input path and directory
	realPath := cleanPath
	if info, err := os.Lstat(cleanPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		// Path exists and is a symlink, resolve it
		if resolved, err := filepath.EvalSymlinks(cleanPath); err == nil {
			realPath = resolved
		}
	}

	// Also resolve symlinks in the configured directory path
	realDir := cleanDir
	if resolved, err := filepath.EvalSymlinks(cleanDir); err == nil {
		realDir = resolved
	}

	// Create directory paths with separators for prefix matching
	dirWithSep := cleanDir
	if !strings.HasSuffix(dirWithSep, string(filepath.Separator)) {
		dirWithSep += string(filepath.Separator)
	}

	realDirWithSep := realDir
	if !strings.HasSuffix(realDirWithSep, string(filepath.Separator)) {
		realDirWithSep += string(filepath.Separator)
	}

	// Check both the original path and the real path against both directory versions
	pathOk := strings.HasPrefix(cleanPath, dirWithSep) || cleanPath == cleanDir ||
		strings.HasPrefix(cleanPath, realDirWithSep) || cleanPath == realDir
	realPathOk := strings.HasPrefix(realPath, dirWithSep) || realPath == cleanDir ||
		strings.HasPrefix(realPath, realDirWithSep) || realPath == realDir

	return pathOk && realPathOk, nil
}

// GetConfiguredDirectory returns the configured directory path
func (v *PathValidator) GetConfiguredDirectory() string {
	return v.configuredDirectory
}

// NormalizePath returns a normalized, absolute path within the configured directory
func (v *PathValidator) NormalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// If path is relative, make it relative to configured directory
	if !filepath.IsAbs(path) {
		path = filepath.Join(v.configuredDirectory, path)
	}

	// Clean and resolve the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate the normalized path
	if err := v.ValidatePath(absPath); err != nil {
		return "", err
	}

	return absPath, nil
}

// ValidateDirectory checks if a directory path is within the configured directory
func (v *PathValidator) ValidateDirectory(dirPath string) error {
	if err := v.ValidatePath(dirPath); err != nil {
		return err
	}

	// If configured directory doesn't exist yet, skip validation
	if _, err := os.Stat(v.configuredDirectory); os.IsNotExist(err) {
		return nil
	}

	// Check if path exists and is a directory
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist yet, which is okay
			return nil
		}
		return fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	return nil
}

// SanitizePath removes potentially dangerous characters and validates the path
func (v *PathValidator) SanitizePath(path string) (string, error) {
	// Remove null bytes and other dangerous characters
	path = strings.ReplaceAll(path, "\x00", "")

	// Normalize the path
	normalized, err := v.NormalizePath(path)
	if err != nil {
		return "", err
	}

	return normalized, nil
}
