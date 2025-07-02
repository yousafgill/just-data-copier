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

	// Create structured logger without source file/line information
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false, // Remove file names and line numbers
	}

	// Use text handler for better console readability
	handler := slog.NewTextHandler(multiWriter, opts)
	logger := slog.New(handler)

	// Set as default logger
	slog.SetDefault(logger)

	slog.Info("Logging initialized", "session_id", time.Now().Format("20060102_150405"))
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
			"max_file_size_mb", "unlimited",
			"buffer_size_kb", float64(cfg.BufferSize)/1024,
			"timeout_seconds", int(cfg.Timeout.Seconds()))
	} else {
		// Get file size if file exists, but don't log the path
		var fileSizeMB float64
		if fileInfo, err := os.Stat(cfg.FilePath); err == nil {
			fileSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
		}

		slog.Info("Client configuration",
			"server_address", cfg.ServerAddress,
			"file_size_mb", fileSizeMB,
			"estimated_chunks", int64(fileSizeMB*1024*1024)/cfg.ChunkSize)
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
			"error_type", "network")
	case *errors.FileSystemError:
		slog.Error("File system error",
			"context", context,
			"operation", e.Op,
			"error_type", "filesystem")
	case *errors.ProtocolError:
		slog.Error("Protocol error",
			"context", context,
			"operation", e.Op,
			"message", e.Message,
			"error_type", "protocol")
	case *errors.CompressionError:
		slog.Error("Compression error",
			"context", context,
			"operation", e.Op,
			"error_type", "compression")
	case *errors.ValidationError:
		slog.Error("Validation error",
			"context", context,
			"field", e.Field,
			"message", e.Message,
			"error_type", "validation")
	default:
		slog.Error("Unhandled error",
			"context", context,
			"error_type", "unknown")
	}
}

// LogTransferProgress logs transfer progress information
func LogTransferProgress(filename string, transferred, total int64, rate float64) {
	percent := float64(transferred) / float64(total) * 100
	slog.Info("Transfer progress",
		"transferred_mb", float64(transferred)/(1024*1024),
		"total_mb", float64(total)/(1024*1024),
		"percent_complete", percent,
		"transfer_rate_mbps", rate,
		"remaining_mb", float64(total-transferred)/(1024*1024))
}

// LogTransferComplete logs successful transfer completion
func LogTransferComplete(filename string, size int64, duration time.Duration) {
	rate := float64(size) / (1024 * 1024) / duration.Seconds()
	slog.Info("Transfer completed successfully",
		"total_size_mb", float64(size)/(1024*1024),
		"duration_seconds", int(duration.Seconds()),
		"average_rate_mbps", rate,
		"timestamp", time.Now().Format("15:04:05"))
}

// LogChunkTransfer logs individual chunk transfer information
func LogChunkTransfer(chunkNum int64, chunkSize int64, totalChunks int64, rate float64) {
	slog.Debug("Chunk transfer",
		"chunk_number", chunkNum,
		"chunk_size_kb", float64(chunkSize)/1024,
		"total_chunks", totalChunks,
		"chunk_rate_mbps", rate,
		"progress_percent", float64(chunkNum)/float64(totalChunks)*100)
}

// LogNetworkMetrics logs network performance metrics
func LogNetworkMetrics(rtt time.Duration, bandwidth int64, packetLoss float64) {
	slog.Info("Network metrics",
		"round_trip_time_ms", rtt.Milliseconds(),
		"estimated_bandwidth_mbps", float64(bandwidth)/(1024*1024),
		"packet_loss_percent", packetLoss*100,
		"network_quality", getNetworkQuality(rtt, packetLoss))
}

// LogSessionStart logs the start of a transfer session
func LogSessionStart(mode string, totalSize int64, chunkSize int64, workers int) {
	totalChunks := (totalSize + chunkSize - 1) / chunkSize // Ceiling division
	slog.Info("Transfer session started",
		"mode", mode,
		"total_size_mb", float64(totalSize)/(1024*1024),
		"chunk_size_kb", float64(chunkSize)/1024,
		"total_chunks", totalChunks,
		"worker_threads", workers,
		"session_start", time.Now().Format("15:04:05"))
}

// LogSessionEnd logs the end of a transfer session
func LogSessionEnd(success bool, totalBytes int64, duration time.Duration) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}

	avgRate := float64(totalBytes) / (1024 * 1024) / duration.Seconds()
	slog.Info("Transfer session ended",
		"status", status,
		"total_bytes_transferred", totalBytes,
		"session_duration_seconds", int(duration.Seconds()),
		"average_throughput_mbps", avgRate,
		"session_end", time.Now().Format("15:04:05"))
}

// getNetworkQuality determines network quality based on metrics
func getNetworkQuality(rtt time.Duration, packetLoss float64) string {
	if rtt < 10*time.Millisecond && packetLoss < 0.001 {
		return "excellent"
	} else if rtt < 50*time.Millisecond && packetLoss < 0.01 {
		return "good"
	} else if rtt < 150*time.Millisecond && packetLoss < 0.05 {
		return "fair"
	}
	return "poor"
}
