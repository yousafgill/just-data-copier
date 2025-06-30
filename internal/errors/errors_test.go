package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationError(t *testing.T) {
	field := "test_field"
	value := "test_value"
	reason := "invalid format"

	err := NewValidationError(field, value, reason)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), field)
	assert.Contains(t, err.Error(), value)
	assert.Contains(t, err.Error(), reason)
	assert.Contains(t, err.Error(), "validation error")
}

func TestNetworkError(t *testing.T) {
	operation := "connect"
	address := "localhost:8000"
	cause := errors.New("connection refused")

	err := NewNetworkError(operation, address, cause)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), operation)
	assert.Contains(t, err.Error(), address)
	assert.Contains(t, err.Error(), cause.Error())
	assert.Contains(t, err.Error(), "network error")
}

func TestFileSystemError(t *testing.T) {
	operation := "read"
	path := "/test/file.txt"
	cause := errors.New("file not found")

	err := NewFileSystemError(operation, path, cause)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), operation)
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), cause.Error())
	assert.Contains(t, err.Error(), "file system error")
}

func TestProtocolError(t *testing.T) {
	operation := "command_read"
	message := "invalid command"
	cause := errors.New("unknown command byte")

	err := NewProtocolError(operation, message, cause)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), operation)
	assert.Contains(t, err.Error(), message)
	assert.Contains(t, err.Error(), cause.Error())
	assert.Contains(t, err.Error(), "protocol error")
}

func TestCompressionError(t *testing.T) {
	operation := "compress"
	cause := errors.New("compression failed")

	err := NewCompressionError(operation, cause)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), operation)
	assert.Contains(t, err.Error(), cause.Error())
	assert.Contains(t, err.Error(), "compression error")
}
