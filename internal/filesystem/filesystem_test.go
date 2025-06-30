package filesystem

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDirectoryExists(t *testing.T) {
	tmpDir := os.TempDir()
	testDir := filepath.Join(tmpDir, "jdc_test_dir")

	// Clean up after test
	defer os.RemoveAll(testDir)

	// Test creating new directory
	err := EnsureDirectoryExists(testDir)
	assert.NoError(t, err)

	// Verify directory exists
	info, err := os.Stat(testDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Test with existing directory
	err = EnsureDirectoryExists(testDir)
	assert.NoError(t, err)
}

func TestGetFileInfo(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "jdc_test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write test content
	content := "test content for file info"
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	// Test getting file info
	info, err := GetFileInfo(tmpFile.Name())
	assert.NoError(t, err)
	assert.False(t, info.IsDir)
	assert.Equal(t, int64(len(content)), info.Size)
	assert.NotZero(t, info.Modified)

	// Test with non-existent file
	_, err = GetFileInfo("non_existent_file.txt")
	assert.Error(t, err)
}

func TestCalculateFileHash(t *testing.T) {
	// Create temporary file with known content
	tmpFile, err := os.CreateTemp("", "jdc_test_hash_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "test content for hash calculation"
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	// Calculate hash using our function
	hash, err := CalculateFileHash(tmpFile)
	assert.NoError(t, err)
	tmpFile.Close()

	// Calculate expected hash
	expected := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	assert.Equal(t, expected, hash)
}

func TestShouldCompress(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.txt", true},
		{"test.log", true},
		{"test.json", true},
		{"test.zip", false},
		{"test.jpg", false},
		{"test.mp4", false},
		{"test.xyz", false}, // unknown extension
	}

	for _, test := range tests {
		result := ShouldCompress(test.filename)
		assert.Equal(t, test.expected, result, "Filename: %s", test.filename)
	}
}

func TestValidateFilePath(t *testing.T) {
	// Test valid paths
	assert.NoError(t, ValidateFilePath("test.txt"))
	assert.NoError(t, ValidateFilePath("dir/test.txt"))

	// Test invalid paths with directory traversal
	// These should still contain ".." after filepath.Clean()
	assert.Error(t, ValidateFilePath("../test.txt"))
	assert.Error(t, ValidateFilePath("dir/../../test.txt"))
}
