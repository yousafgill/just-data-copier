package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid server config",
			config: Config{
				IsServer:      true,
				ListenAddress: "localhost:8000",
				OutputDir:     "./output",
				ChunkSize:     1024 * 1024,
				BufferSize:    512 * 1024,
				Workers:       4,
				Timeout:       time.Minute,
				Retries:       3,
			},
			wantErr: false,
		},
		{
			name: "valid client config",
			config: Config{
				IsServer:      false,
				ServerAddress: "localhost:8000",
				FilePath:      "test.txt",
				ChunkSize:     1024 * 1024,
				BufferSize:    512 * 1024,
				Workers:       4,
				Timeout:       time.Minute,
				Retries:       3,
			},
			wantErr: false,
		},
		{
			name: "invalid chunk size",
			config: Config{
				ChunkSize:  0,
				BufferSize: 512 * 1024,
				Workers:    4,
				Timeout:    time.Minute,
				Retries:    3,
			},
			wantErr: true,
			errMsg:  "chunk size must be positive",
		},
		{
			name: "invalid buffer size",
			config: Config{
				ChunkSize:  1024 * 1024,
				BufferSize: 0,
				Workers:    4,
				Timeout:    time.Minute,
				Retries:    3,
			},
			wantErr: true,
			errMsg:  "buffer size must be positive",
		},
		{
			name: "invalid workers",
			config: Config{
				ChunkSize:  1024 * 1024,
				BufferSize: 512 * 1024,
				Workers:    0,
				Timeout:    time.Minute,
				Retries:    3,
			},
			wantErr: true,
			errMsg:  "workers must be positive",
		},
		{
			name: "negative retries",
			config: Config{
				ChunkSize:  1024 * 1024,
				BufferSize: 512 * 1024,
				Workers:    4,
				Timeout:    time.Minute,
				Retries:    -1,
			},
			wantErr: true,
			errMsg:  "retries cannot be negative",
		},
		{
			name: "invalid timeout",
			config: Config{
				ChunkSize:  1024 * 1024,
				BufferSize: 512 * 1024,
				Workers:    4,
				Timeout:    0,
				Retries:    3,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "client without file path",
			config: Config{
				IsServer:   false,
				ChunkSize:  1024 * 1024,
				BufferSize: 512 * 1024,
				Workers:    4,
				Timeout:    time.Minute,
				Retries:    3,
			},
			wantErr: true,
			errMsg:  "file path is required in client mode",
		},
		{
			name: "invalid adaptive delay config",
			config: Config{
				ChunkSize:     1024 * 1024,
				BufferSize:    512 * 1024,
				Workers:       4,
				Timeout:       time.Minute,
				Retries:       3,
				AdaptiveDelay: true,
				MinDelay:      100 * time.Millisecond,
				MaxDelay:      50 * time.Millisecond, // Max < Min
			},
			wantErr: true,
			errMsg:  "invalid adaptive delay configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "server config",
			config: Config{
				IsServer:    true,
				ChunkSize:   1024 * 1024,
				BufferSize:  512 * 1024,
				Workers:     4,
				Compression: true,
			},
			expected: "Config{Mode: Server, ChunkSize: 1048576, BufferSize: 524288, Workers: 4, Compression: true, AdaptiveDelay: false}",
		},
		{
			name: "client config",
			config: Config{
				IsServer:      false,
				ChunkSize:     2 * 1024 * 1024,
				BufferSize:    1024 * 1024,
				Workers:       8,
				Compression:   false,
				AdaptiveDelay: true,
			},
			expected: "Config{Mode: Client, ChunkSize: 2097152, BufferSize: 1048576, Workers: 8, Compression: false, AdaptiveDelay: true}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
