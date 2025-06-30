package config

import (
	"flag"
	"fmt"
	"runtime"
	"time"
)

// Constants for default values
const (
	DefaultChunkSize  = 2 * 1024 * 1024 // 2MB
	DefaultBufferSize = 512 * 1024      // 512KB
	DefaultTimeout    = 2 * time.Minute
	DefaultRetries    = 5
	DefaultChunkDelay = 10 * time.Millisecond
	DefaultMinDelay   = 1 * time.Millisecond
	DefaultMaxDelay   = 100 * time.Millisecond
	DefaultListenAddr = "0.0.0.0:8000"
	DefaultServerAddr = "localhost:8000"
	DefaultOutputDir  = "./output"

	// Buffer size constants
	SmallWriteSize  = 8 * 1024   // 8KB
	MediumWriteSize = 32 * 1024  // 32KB
	LargeWriteSize  = 64 * 1024  // 64KB
	MaxWriteSize    = 256 * 1024 // 256KB

	// Network constants
	TCPBufferSize  = 1024 * 1024     // 1MB
	HashBufferSize = 4 * 1024 * 1024 // 4MB
	ProfileTimeout = 5 * time.Second
	PingCount      = 5

	// File system constants
	StateFileExt   = ".justdatacopier.state"
	LogDirPerms    = 0755
	StateFilePerms = 0644
)

// Config holds all configuration parameters for the application
type Config struct {
	// Server mode settings
	IsServer      bool
	ListenAddress string
	OutputDir     string

	// Client mode settings
	ServerAddress string
	FilePath      string

	// Common parameters
	ChunkSize     int64
	BufferSize    int
	Workers       int
	Compression   bool
	VerifyHash    bool
	ShowProgress  bool
	Timeout       time.Duration
	Retries       int
	ChunkDelay    time.Duration
	AdaptiveDelay bool
	MinDelay      time.Duration
	MaxDelay      time.Duration
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ChunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive")
	}
	if c.BufferSize <= 0 {
		return fmt.Errorf("buffer size must be positive")
	}
	if c.Workers <= 0 {
		return fmt.Errorf("workers must be positive")
	}
	if c.Retries < 0 {
		return fmt.Errorf("retries cannot be negative")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.AdaptiveDelay && (c.MinDelay <= 0 || c.MaxDelay <= 0 || c.MinDelay > c.MaxDelay) {
		return fmt.Errorf("invalid adaptive delay configuration")
	}

	if !c.IsServer && c.FilePath == "" {
		return fmt.Errorf("file path is required in client mode")
	}

	return nil
}

// ParseFlags parses command line arguments and returns a Config
func ParseFlags() (*Config, error) {
	// Server flags
	isServer := flag.Bool("server", false, "Run in server mode")
	listenAddr := flag.String("listen", DefaultListenAddr, "Address to listen on (server mode)")
	outputDir := flag.String("output", DefaultOutputDir, "Directory to store received files (server mode)")

	// Client flags
	serverAddr := flag.String("connect", DefaultServerAddr, "Server address to connect to (client mode)")
	filePath := flag.String("file", "", "File to transfer (client mode)")

	// Common flags
	chunkSize := flag.Int64("chunk", DefaultChunkSize, "Chunk size in bytes (2MB default)")
	bufferSize := flag.Int("buffer", DefaultBufferSize, "Buffer size in bytes (512KB default)")
	workers := flag.Int("workers", runtime.NumCPU()/2, "Number of worker threads")
	compression := flag.Bool("compress", false, "Enable gzip compression")
	verifyHash := flag.Bool("verify", true, "Verify file integrity with MD5")
	showProgress := flag.Bool("progress", true, "Show progress during transfer")
	timeout := flag.Duration("timeout", DefaultTimeout, "Operation timeout")
	retries := flag.Int("retries", DefaultRetries, "Number of retries for failed operations")
	chunkDelay := flag.Duration("delay", DefaultChunkDelay, "Delay between chunk transfers")
	adaptiveDelay := flag.Bool("adaptive", false, "Use adaptive delay based on network conditions")
	minDelay := flag.Duration("min-delay", DefaultMinDelay, "Minimum delay for adaptive networking")
	maxDelay := flag.Duration("max-delay", DefaultMaxDelay, "Maximum delay for adaptive networking")

	flag.Parse()

	config := &Config{
		IsServer:      *isServer,
		ListenAddress: *listenAddr,
		OutputDir:     *outputDir,
		ServerAddress: *serverAddr,
		FilePath:      *filePath,
		ChunkSize:     *chunkSize,
		BufferSize:    *bufferSize,
		Workers:       *workers,
		Compression:   *compression,
		VerifyHash:    *verifyHash,
		ShowProgress:  *showProgress,
		Timeout:       *timeout,
		Retries:       *retries,
		ChunkDelay:    *chunkDelay,
		AdaptiveDelay: *adaptiveDelay,
		MinDelay:      *minDelay,
		MaxDelay:      *maxDelay,
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// String returns a string representation of the config for logging
func (c *Config) String() string {
	mode := "Client"
	if c.IsServer {
		mode = "Server"
	}

	return fmt.Sprintf("Config{Mode: %s, ChunkSize: %d, BufferSize: %d, Workers: %d, Compression: %v, AdaptiveDelay: %v}",
		mode, c.ChunkSize, c.BufferSize, c.Workers, c.Compression, c.AdaptiveDelay)
}
