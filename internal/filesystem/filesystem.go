package filesystem

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"justdatacopier/internal/config"
	"justdatacopier/internal/errors"
)

// TransferState represents the state of a file transfer for resume capability
type TransferState struct {
	Filename       string    `json:"filename"`
	FileSize       int64     `json:"file_size"`
	ChunkSize      int64     `json:"chunk_size"`
	NumChunks      int64     `json:"num_chunks"`
	ChunksReceived []bool    `json:"chunks_received"`
	LastModified   time.Time `json:"last_modified"`
	Version        int       `json:"version"`
}

// FileInfo represents information about a file to be transferred
type FileInfo struct {
	Name     string
	Size     int64
	Path     string
	IsDir    bool
	Modified time.Time
}

// ValidateFilePath checks if a file path is safe and valid
func ValidateFilePath(path string) error {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return errors.NewValidationError("file_path", path, "path contains directory traversal")
	}

	// Check for absolute paths in client mode (security concern)
	if filepath.IsAbs(cleanPath) && strings.Contains(path, ":") {
		// Allow absolute paths but log them
		slog.Warn("Absolute path detected", "path", path)
	}

	return nil
}

// GetFileInfo returns information about a file
func GetFileInfo(path string) (*FileInfo, error) {
	if err := ValidateFilePath(path); err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, errors.NewFileSystemError("stat", path, err)
	}

	return &FileInfo{
		Name:     stat.Name(),
		Size:     stat.Size(),
		Path:     path,
		IsDir:    stat.IsDir(),
		Modified: stat.ModTime(),
	}, nil
}

// SaveTransferState saves the current transfer state to disk
func SaveTransferState(state *TransferState, outputDir string) error {
	stateFile := filepath.Join(outputDir, state.Filename+config.StateFileExt)

	state.Version = 1 // Set version for future compatibility
	state.LastModified = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.NewFileSystemError("marshal_state", stateFile, err)
	}

	if err := os.WriteFile(stateFile, data, config.StateFilePerms); err != nil {
		return errors.NewFileSystemError("write_state", stateFile, err)
	}

	return nil
}

// LoadTransferState loads transfer state from disk
func LoadTransferState(filename, outputDir string) (*TransferState, error) {
	stateFile := filepath.Join(outputDir, filename+config.StateFileExt)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, errors.NewFileSystemError("read_state", stateFile, err)
	}

	var state TransferState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, errors.NewFileSystemError("unmarshal_state", stateFile, err)
	}

	// Version compatibility check
	if state.Version == 0 {
		state.Version = 1 // Upgrade old state files
	}

	return &state, nil
}

// RemoveTransferState removes the transfer state file
func RemoveTransferState(filename, outputDir string) error {
	stateFile := filepath.Join(outputDir, filename+config.StateFileExt)

	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return errors.NewFileSystemError("remove_state", stateFile, err)
	}

	return nil
}

// PreallocateFile preallocates disk space for a file to improve performance
func PreallocateFile(file *os.File, size int64) error {
	// Try to use fallocate on supported systems
	if err := fallocate(file, size); err == nil {
		return nil
	}

	// Fallback to truncate
	if err := file.Truncate(size); err != nil {
		return errors.NewFileSystemError("truncate", file.Name(), err)
	}

	// Reset file position
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return errors.NewFileSystemError("seek", file.Name(), err)
	}

	return nil
}

// fallocate attempts to use fallocate on supported systems
func fallocate(file *os.File, size int64) error {
	// On Windows, fallocate is not available, so we return an error
	// to fall back to truncate
	return errors.NewFileSystemError("fallocate", file.Name(),
		errors.NewValidationError("system", "windows", "fallocate not supported"))
}

// CalculateFileHash calculates MD5 hash of a file
func CalculateFileHash(file *os.File) (string, error) {
	// Reset file position
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", errors.NewFileSystemError("seek", file.Name(), err)
	}

	hash := md5.New()
	buffer := make([]byte, config.HashBufferSize)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.NewFileSystemError("read_hash", file.Name(), err)
		}

		if _, err := hash.Write(buffer[:n]); err != nil {
			return "", errors.NewFileSystemError("hash_write", file.Name(), err)
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(dir string) error {
	if err := ValidateFilePath(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, config.LogDirPerms); err != nil {
		return errors.NewFileSystemError("mkdir", dir, err)
	}

	return nil
}

// GetCompressibleExtensions returns a map of file extensions that should be compressed
func GetCompressibleExtensions() map[string]bool {
	return map[string]bool{
		".txt":  true,
		".log":  true,
		".csv":  true,
		".json": true,
		".xml":  true,
		".html": true,
		".htm":  true,
		".css":  true,
		".js":   true,
		".sql":  true,
		".md":   true,
		".yaml": true,
		".yml":  true,
		".ini":  true,
		".conf": true,
		".cfg":  true,
	}
}

// GetAlreadyCompressedExtensions returns a map of file extensions that are already compressed
func GetAlreadyCompressedExtensions() map[string]bool {
	return map[string]bool{
		".zip":  true,
		".gz":   true,
		".bz2":  true,
		".xz":   true,
		".rar":  true,
		".7z":   true,
		".tar":  true,
		".mp3":  true,
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".pdf":  true,
		".docx": true,
		".xlsx": true,
		".pptx": true,
		".odt":  true,
		".ods":  true,
		".odp":  true,
	}
}

// ShouldCompress determines if a file should be compressed based on its extension
func ShouldCompress(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	// Don't compress already compressed files
	if GetAlreadyCompressedExtensions()[ext] {
		return false
	}

	// Compress known compressible files
	if GetCompressibleExtensions()[ext] {
		return true
	}

	// Default to no compression for unknown extensions
	return false
}

// SafeFileOperation performs a file operation with proper error handling
func SafeFileOperation(op string, fn func() error) error {
	if err := fn(); err != nil {
		return errors.NewFileSystemError(op, "", err)
	}
	return nil
}
