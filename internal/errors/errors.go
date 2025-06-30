package errors

import (
	"errors"
	"fmt"
)

// Error types for different categories of failures
var (
	ErrNetwork     = errors.New("network error")
	ErrFileSystem  = errors.New("file system error")
	ErrProtocol    = errors.New("protocol error")
	ErrCompression = errors.New("compression error")
	ErrValidation  = errors.New("validation error")
	ErrTimeout     = errors.New("timeout error")
	ErrCancelled   = errors.New("operation cancelled")
)

// NetworkError represents network-related errors
type NetworkError struct {
	Op   string
	Addr string
	Err  error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s to %s: %v", e.Op, e.Addr, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

func (e *NetworkError) Is(target error) bool {
	return target == ErrNetwork
}

// FileSystemError represents file system-related errors
type FileSystemError struct {
	Op   string
	Path string
	Err  error
}

func (e *FileSystemError) Error() string {
	return fmt.Sprintf("file system error during %s on %s: %v", e.Op, e.Path, e.Err)
}

func (e *FileSystemError) Unwrap() error {
	return e.Err
}

func (e *FileSystemError) Is(target error) bool {
	return target == ErrFileSystem
}

// ProtocolError represents protocol-related errors
type ProtocolError struct {
	Op      string
	Message string
	Err     error
}

func (e *ProtocolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("protocol error during %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("protocol error during %s: %s", e.Op, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}

func (e *ProtocolError) Is(target error) bool {
	return target == ErrProtocol
}

// CompressionError represents compression-related errors
type CompressionError struct {
	Op  string
	Err error
}

func (e *CompressionError) Error() string {
	return fmt.Sprintf("compression error during %s: %v", e.Op, e.Err)
}

func (e *CompressionError) Unwrap() error {
	return e.Err
}

func (e *CompressionError) Is(target error) bool {
	return target == ErrCompression
}

// ValidationError represents validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s='%v': %s", e.Field, e.Value, e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

// Helper functions for creating errors

func NewNetworkError(op, addr string, err error) error {
	return &NetworkError{Op: op, Addr: addr, Err: err}
}

func NewFileSystemError(op, path string, err error) error {
	return &FileSystemError{Op: op, Path: path, Err: err}
}

func NewProtocolError(op, message string, err error) error {
	return &ProtocolError{Op: op, Message: message, Err: err}
}

func NewCompressionError(op string, err error) error {
	return &CompressionError{Op: op, Err: err}
}

func NewValidationError(field string, value interface{}, message string) error {
	return &ValidationError{Field: field, Value: value, Message: message}
}
