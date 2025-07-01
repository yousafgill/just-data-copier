package client

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"justdatacopier/internal/compression"
	"justdatacopier/internal/config"
	"justdatacopier/internal/errors"
	"justdatacopier/internal/filesystem"
	"justdatacopier/internal/logging"
	"justdatacopier/internal/network"
	"justdatacopier/internal/progress"
	"justdatacopier/internal/protocol"
)

// Run starts the client with the given configuration
func Run(cfg *config.Config) error {
	slog.Info("Starting client", "server", cfg.ServerAddress)

	// Get file information
	fileInfo, err := filesystem.GetFileInfo(cfg.FilePath)
	if err != nil {
		return err
	}

	if fileInfo.IsDir {
		return errors.NewValidationError("file_path", cfg.FilePath, "cannot transfer directories")
	}

	// Open file for reading
	file, err := os.Open(cfg.FilePath)
	if err != nil {
		return errors.NewFileSystemError("open", cfg.FilePath, err)
	}
	defer file.Close()

	// Connect to server
	conn, err := net.Dial("tcp", cfg.ServerAddress)
	if err != nil {
		return errors.NewNetworkError("dial", cfg.ServerAddress, err)
	}
	defer conn.Close()

	// Disable connection deadline for persistent connections
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return errors.NewNetworkError("set_deadline", cfg.ServerAddress, err)
	}

	// Apply TCP optimizations
	if err := network.OptimizeTCPConnection(conn); err != nil {
		slog.Warn("Failed to optimize TCP connection", "error", err)
	}

	// Create buffered reader and writer
	reader := bufio.NewReaderSize(conn, cfg.BufferSize)
	writer := bufio.NewWriterSize(conn, cfg.BufferSize)

	// Perform network profiling
	slog.Info("Performing network profiling...")
	profile := network.ProfileNetwork(conn)
	logging.LogNetworkMetrics(profile.RTT, profile.Bandwidth, profile.PacketLoss)

	// Adjust configuration based on profile
	adjustConfigForNetwork(cfg, profile)

	// Re-create reader and writer with optimal buffer sizes
	optimalBufferSize := max(cfg.BufferSize, int(profile.OptimalChunkSize/4))
	reader = bufio.NewReaderSize(conn, optimalBufferSize)
	writer = bufio.NewWriterSize(conn, optimalBufferSize)

	// Initialize transfer
	if err := initializeTransfer(writer, fileInfo, cfg); err != nil {
		return err
	}

	// Setup transfer statistics
	stats := &progress.Stats{
		TotalBytes: fileInfo.Size,
		StartTime:  time.Now(),
		FileSize:   fileInfo.Size,
		Filename:   fileInfo.Name,
	}

	// Setup network statistics
	netStats := network.NewNetworkStats(cfg)

	// Create buffer pool for chunks
	bufferPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, cfg.ChunkSize)
		},
	}

	// Start progress reporting
	var reporter *progress.Reporter
	if cfg.ShowProgress {
		reporter = progress.NewReporter(stats, cfg.ShowProgress)
		reporter.Start()
		defer reporter.Stop()
	}

	// Handle server requests
	return handleServerRequests(reader, writer, file, stats, netStats, &bufferPool, cfg)
}

// adjustConfigForNetwork adjusts configuration based on network profile
func adjustConfigForNetwork(cfg *config.Config, profile network.NetworkProfile) {
	originalWorkers := cfg.Workers
	originalChunkSize := cfg.ChunkSize

	switch {
	case profile.RTT > 100*time.Millisecond:
		// High latency network, reduce workers and increase chunk size
		cfg.Workers = max(1, cfg.Workers/2)
		cfg.ChunkSize = profile.OptimalChunkSize
		slog.Info("High latency network detected",
			"old_workers", originalWorkers,
			"new_workers", cfg.Workers)
	case profile.RTT < 10*time.Millisecond:
		// Low latency network, can use more workers
		cfg.Workers = min(runtime.NumCPU(), cfg.Workers*2)
		slog.Info("Low latency network detected",
			"old_workers", originalWorkers,
			"new_workers", cfg.Workers)
	}

	if cfg.ChunkSize != originalChunkSize {
		slog.Info("Adjusted chunk size based on network",
			"old_size_mb", float64(originalChunkSize)/(1024*1024),
			"new_size_mb", float64(cfg.ChunkSize)/(1024*1024))
	}
}

// initializeTransfer sends initial transfer information to server
func initializeTransfer(writer *bufio.Writer, fileInfo *filesystem.FileInfo, cfg *config.Config) error {
	// Send initialization command
	if err := protocol.SendCommand(writer, protocol.CmdInit); err != nil {
		return err
	}

	// Send filename
	if err := protocol.SendString(writer, fileInfo.Name); err != nil {
		return err
	}

	// Send file size
	if err := protocol.SendInt64(writer, fileInfo.Size); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	logging.LogSessionStart("CLIENT", fileInfo.Size, int64(cfg.ChunkSize), cfg.Workers)

	return nil
}

// handleServerRequests handles requests from the server
func handleServerRequests(reader *bufio.Reader, writer *bufio.Writer, file *os.File,
	stats *progress.Stats, netStats *network.NetworkStats, bufferPool *sync.Pool, cfg *config.Config) error {

	ctx := context.Background()

	for {
		// Read command from server
		cmdByte, err := protocol.ReadCommand(ctx, reader)
		if err != nil {
			return errors.NewNetworkError("read_command", "", err)
		}

		switch cmdByte {
		case protocol.CmdRequest:
			if err := handleChunkRequest(ctx, reader, writer, file, stats, netStats, bufferPool, cfg); err != nil {
				return err
			}

		case protocol.CmdHash:
			if err := handleHashRequest(writer, file); err != nil {
				return err
			}

		case protocol.CmdComplete:
			// Transfer completed successfully
			elapsed := time.Since(stats.StartTime)
			logging.LogTransferComplete(stats.Filename, stats.FileSize, elapsed)
			return nil

		case protocol.CmdError:
			// Read error message from server
			errorMsg, _ := protocol.ReadString(ctx, reader)
			return errors.NewProtocolError("server_error", errorMsg, nil)

		default:
			return errors.NewProtocolError("unknown_command", "unexpected command from server", nil)
		}
	}
}

// handleChunkRequest handles a chunk request from the server
func handleChunkRequest(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer,
	file *os.File, stats *progress.Stats, netStats *network.NetworkStats,
	bufferPool *sync.Pool, cfg *config.Config) error {

	// Read chunk offset
	offset, err := protocol.ReadInt64(ctx, reader)
	if err != nil {
		return err
	}

	// Apply adaptive delay
	chunkDelay := netStats.GetDelay(cfg.ChunkDelay)
	if chunkDelay > 0 {
		time.Sleep(chunkDelay)
	}

	// Get buffer from pool
	buffer := bufferPool.Get().([]byte)
	defer bufferPool.Put(buffer)

	// Calculate actual chunk size (last chunk might be smaller)
	actualChunkSize := cfg.ChunkSize
	if offset+cfg.ChunkSize > stats.FileSize {
		actualChunkSize = stats.FileSize - offset
	}

	// Send chunk data
	if err := sendChunk(writer, file, offset, actualChunkSize, buffer, stats, cfg); err != nil {
		return err
	}

	// Update network stats
	netStats.UpdateStats(actualChunkSize)
	return nil
}

// handleHashRequest handles a hash request from the server
func handleHashRequest(writer *bufio.Writer, file *os.File) error {
	// Calculate file hash
	hash, err := filesystem.CalculateFileHash(file)
	if err != nil {
		return err
	}

	// Send hash command and hash value
	if err := protocol.SendCommand(writer, protocol.CmdHash); err != nil {
		return err
	}

	if err := protocol.SendString(writer, hash); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	slog.Info("File hash sent", "hash", hash)
	return nil
}

// sendChunk sends a chunk of data to the server
func sendChunk(writer *bufio.Writer, file *os.File, offset, chunkSize int64,
	buffer []byte, stats *progress.Stats, cfg *config.Config) error {

	// Read chunk from file
	n, err := file.ReadAt(buffer, offset)
	if err != nil && err != io.EOF {
		return errors.NewFileSystemError("read_chunk", file.Name(), err)
	}

	// Create context with timeout
	timeoutPerMB := 10 * time.Second
	chunkSizeMB := float64(n) / (1024 * 1024)
	chunkTimeout := time.Duration(max(30, int(chunkSizeMB*timeoutPerMB.Seconds()))) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), chunkTimeout)
	defer cancel()

	// Send with retries
	const maxRetries = 5
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Exponential backoff for retries
		if retry > 0 {
			backoffTime := time.Duration(retry*500) * time.Millisecond
			time.Sleep(backoffTime)
			slog.Debug("Retrying chunk send", "offset", offset, "attempt", retry+1)
		}

		err := sendChunkData(ctx, writer, file, buffer[:n], cfg)
		if err == nil {
			stats.UpdateTransferred(int64(n))
			return nil
		}

		lastErr = err
	}

	return errors.NewNetworkError("send_chunk", "", lastErr)
}

// sendChunkData sends the actual chunk data with compression if enabled
func sendChunkData(ctx context.Context, writer *bufio.Writer, file *os.File,
	data []byte, cfg *config.Config) error {

	// Send data command
	if err := protocol.SendCommand(writer, protocol.CmdData); err != nil {
		return err
	}

	// Send chunk size
	if err := protocol.SendInt64(writer, int64(len(data))); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	// Handle compression
	if cfg.Compression && compression.ShouldCompressFile(file.Name()) {
		return sendCompressedChunk(ctx, writer, file.Name(), data)
	}

	return sendUncompressedChunk(ctx, writer, data)
}

// sendCompressedChunk sends data with compression
func sendCompressedChunk(ctx context.Context, writer *bufio.Writer, filename string, data []byte) error {
	// Compress data
	compressedData, err := compression.CompressData(data, filename)
	if err != nil {
		return err
	}

	// Send compression flag (1 = compressed)
	if err := protocol.SendCommand(writer, 1); err != nil {
		return err
	}

	// Send compressed size
	if err := protocol.SendInt64(writer, int64(len(compressedData))); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	// Log compression ratio
	ratio := compression.GetCompressionRatio(len(data), len(compressedData))
	slog.Debug("Chunk compressed",
		"original_size", len(data),
		"compressed_size", len(compressedData),
		"ratio", ratio)

	// Send compressed data in pieces
	return sendDataInPieces(ctx, writer, compressedData)
}

// sendUncompressedChunk sends data without compression
func sendUncompressedChunk(ctx context.Context, writer *bufio.Writer, data []byte) error {
	// Send compression flag (0 = uncompressed)
	if err := protocol.SendCommand(writer, 0); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	// Send uncompressed data in pieces
	return sendDataInPieces(ctx, writer, data)
}

// sendDataInPieces sends data in smaller pieces with adaptive sizing
func sendDataInPieces(ctx context.Context, writer *bufio.Writer, data []byte) error {
	maxWriteSize := config.LargeWriteSize // Start with 64KB chunks
	minWriteSize := config.SmallWriteSize // Don't go below 8KB
	consecutiveSlowWrites := 0

	for i := 0; i < len(data); {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate piece size
		endPos := i + maxWriteSize
		if endPos > len(data) {
			endPos = len(data)
		}

		// Track write time for adaptive sizing
		writeStart := time.Now()

		// Write piece
		if _, err := writer.Write(data[i:endPos]); err != nil {
			return errors.NewNetworkError("write_piece", "", err)
		}

		if err := protocol.FlushWriter(writer); err != nil {
			return errors.NewNetworkError("flush_piece", "", err)
		}

		// Adapt write size based on performance
		pieceTime := time.Since(writeStart)

		if pieceTime > 2*time.Second {
			// Too slow, reduce size
			maxWriteSize = max(minWriteSize, maxWriteSize/2)
			consecutiveSlowWrites++

			slog.Debug("Slow network detected, reducing write size",
				"piece_time", pieceTime,
				"new_size", maxWriteSize)

			// Pause if multiple slow writes
			if consecutiveSlowWrites > 2 {
				pauseTime := 500 * time.Millisecond * time.Duration(consecutiveSlowWrites-2)
				if pauseTime > 5*time.Second {
					pauseTime = 5 * time.Second
				}
				time.Sleep(pauseTime)
			}
		} else if pieceTime < 200*time.Millisecond && maxWriteSize < config.MaxWriteSize {
			// Fast enough, try increasing size
			maxWriteSize = min(maxWriteSize*2, config.MaxWriteSize)
			consecutiveSlowWrites = 0
		} else {
			consecutiveSlowWrites = 0
		}

		i = endPos
	}

	return nil
}

// Helper functions for min/max operations
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
