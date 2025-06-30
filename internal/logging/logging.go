package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"justdatacopier/internal/config"
	"justdatacopier/internal/errors"
	"justdatacopier/internal/filesystem"
)

// SetupLogger initializes structured logging with file and console output
func SetupLogger() error {
	// Create logs directory if it doesn't exist
	if err := filesystem.EnsureDirectoryExists("logs"); err != nil {
		return err
	}

	// Create log file with timestamp
	logFileName := filepath.Join("logs",
		"justdatacopier_"+time.Now().Format("20060102_150405")+".log")

	logFile, err := os.Create(logFileName)
	if err != nil {
		// Continue with console logging only
		slog.Warn("Failed to create log file, using console only", "error", err)
		return nil
	}

	// Create multi-writer to log to both console and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create structured logger with JSON format for file, text for console
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	// Use text handler for better console readability
	handler := slog.NewTextHandler(multiWriter, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	slog.Info("Logging initialized", "log_file", logFileName)
	return nil
}

// LogConfig logs the current configuration
func LogConfig(cfg *config.Config) {
	mode := "Client"
	if cfg.IsServer {
		mode = "Server"
	}

	slog.Info("Configuration loaded",
		"mode", mode,
		"chunk_size_mb", float64(cfg.ChunkSize)/(1024*1024),
		"buffer_size_kb", float64(cfg.BufferSize)/1024,
		"workers", cfg.Workers,
		"compression", cfg.Compression,
		"adaptive_delay", cfg.AdaptiveDelay,
		"verify_hash", cfg.VerifyHash)

	if cfg.IsServer {
		slog.Info("Server configuration",
			"listen_address", cfg.ListenAddress,
			"output_directory", cfg.OutputDir)
	} else {
		slog.Info("Client configuration",
			"server_address", cfg.ServerAddress,
			"file_path", cfg.FilePath)
	}
}

// LogError logs an error with appropriate context
func LogError(err error, context string) {
	switch e := err.(type) {
	case *errors.NetworkError:
		slog.Error("Network error",
			"context", context,
			"operation", e.Op,
			"address", e.Addr,
			"error", e.Err)
	case *errors.FileSystemError:
		slog.Error("File system error",
			"context", context,
			"operation", e.Op,
			"path", e.Path,
			"error", e.Err)
	case *errors.ProtocolError:
		slog.Error("Protocol error",
			"context", context,
			"operation", e.Op,
			"message", e.Message,
			"error", e.Err)
	case *errors.CompressionError:
		slog.Error("Compression error",
			"context", context,
			"operation", e.Op,
			"error", e.Err)
	case *errors.ValidationError:
		slog.Error("Validation error",
			"context", context,
			"field", e.Field,
			"value", e.Value,
			"message", e.Message)
	default:
		slog.Error("Unhandled error",
			"context", context,
			"error", err)
	}
}

// LogTransferProgress logs transfer progress information
func LogTransferProgress(filename string, transferred, total int64, rate float64) {
	percent := float64(transferred) / float64(total) * 100
	slog.Info("Transfer progress",
		"filename", filename,
		"transferred_mb", float64(transferred)/(1024*1024),
		"total_mb", float64(total)/(1024*1024),
		"percent", percent,
		"rate_mbps", rate)
}

// LogTransferComplete logs successful transfer completion
func LogTransferComplete(filename string, size int64, duration time.Duration) {
	rate := float64(size) / (1024 * 1024) / duration.Seconds()
	slog.Info("Transfer completed successfully",
		"filename", filename,
		"size_mb", float64(size)/(1024*1024),
		"duration", duration.Round(time.Second),
		"average_rate_mbps", rate)
}
