package server

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
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

// Run starts the server with the given configuration
func Run(cfg *config.Config) error {
	slog.Info("Starting server", "address", cfg.ListenAddress, "workers", cfg.Workers)

	// Create output directory if it doesn't exist
	if err := filesystem.EnsureDirectoryExists(cfg.OutputDir); err != nil {
		return err
	}

	// Start listener
	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		return errors.NewNetworkError("listen", cfg.ListenAddress, err)
	}
	defer listener.Close()

	slog.Info("Server ready to accept connections")

	// Accept and handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept connection", "error", err)
			continue
		}

		go handleConnection(conn, cfg)
	}
}

// handleConnection handles a single client connection
func handleConnection(conn net.Conn, cfg *config.Config) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	slog.Info("New connection", "remote_addr", remoteAddr)

	// Disable connection deadline for persistent connections
	if err := conn.SetDeadline(time.Time{}); err != nil {
		slog.Error("Failed to disable connection deadline", "error", err)
		return
	}

	// Apply TCP optimizations
	if err := network.OptimizeTCPConnection(conn); err != nil {
		slog.Warn("Failed to optimize TCP connection", "error", err)
	}

	// Create buffered reader and writer
	reader := bufio.NewReaderSize(conn, cfg.BufferSize)
	writer := bufio.NewWriterSize(conn, cfg.BufferSize)

	// Handle commands in a loop
	for {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)

		cmdByte, err := protocol.ReadCommand(ctx, reader)
		cancel()

		if err != nil {
			if err == io.EOF {
				slog.Info("Connection closed by client", "remote_addr", remoteAddr)
			} else {
				slog.Error("Failed to read command", "error", err)
			}
			return
		}

		switch cmdByte {
		case protocol.CmdInit:
			handleFileTransfer(reader, writer, conn, cfg)
			return // Close connection after file transfer
		case protocol.CmdPing:
			handlePing(writer)
		default:
			slog.Error("Unknown command", "command", cmdByte)
			protocol.SendError(writer, "Unknown command")
			return
		}
	}
}

// handlePing responds to ping requests for network profiling
func handlePing(writer *bufio.Writer) {
	if err := protocol.SendCommand(writer, protocol.CmdPong); err != nil {
		slog.Error("Failed to send pong", "error", err)
		return
	}

	if err := protocol.FlushWriter(writer); err != nil {
		slog.Error("Failed to flush pong", "error", err)
	}
}

// handleFileTransfer handles the complete file transfer process
func handleFileTransfer(reader *bufio.Reader, writer *bufio.Writer, conn net.Conn, cfg *config.Config) {
	ctx := context.Background()

	// Read filename
	filename, err := protocol.ReadString(ctx, reader)
	if err != nil {
		slog.Error("Failed to read filename", "error", err)
		protocol.SendError(writer, "Failed to read filename")
		return
	}

	baseFilename := filepath.Base(filename)
	slog.Info("Receiving file", "file_size_mb", "pending")

	// Read file size
	fileSize, err := protocol.ReadInt64(ctx, reader)
	if err != nil {
		slog.Error("Failed to read file size", "error", err)
		protocol.SendError(writer, "Failed to read file size")
		return
	}

	// Validate file size
	if fileSize <= 0 {
		slog.Error("Invalid file size", "size", fileSize)
		protocol.SendError(writer, "Invalid file size")
		return
	}

	logging.LogSessionStart("SERVER", fileSize, cfg.ChunkSize, cfg.Workers)

	// Setup transfer state
	outputPath := filepath.Join(cfg.OutputDir, baseFilename)
	numChunks := (fileSize + cfg.ChunkSize - 1) / cfg.ChunkSize

	// Try to resume existing transfer
	transferState, resuming := tryResumeTransfer(baseFilename, cfg, fileSize, numChunks)

	// Create or open output file
	outFile, err := createOrOpenOutputFile(outputPath, resuming)
	if err != nil {
		slog.Error("Failed to create output file", "error", err)
		protocol.SendError(writer, "File creation failed")
		return
	}
	defer outFile.Close()

	// Pre-allocate file space if not resuming
	if !resuming {
		if err := filesystem.PreallocateFile(outFile, fileSize); err != nil {
			slog.Warn("Failed to preallocate file space", "error", err)
		}
	}

	// Initialize progress tracking
	stats := &progress.Stats{
		TotalBytes: fileSize,
		StartTime:  time.Now(),
		FileSize:   fileSize,
		Filename:   baseFilename,
	}

	if resuming {
		resumeOffset := calculateResumeOffset(transferState)
		stats.SetTransferred(resumeOffset)
		slog.Info("Resuming transfer", "offset_mb", float64(resumeOffset)/(1024*1024))
	}

	// Start progress reporting
	var reporter *progress.Reporter
	if cfg.ShowProgress {
		reporter = progress.NewReporter(stats, cfg.ShowProgress)
		reporter.Start()
		defer reporter.Stop()
	}

	// Setup network statistics
	netStats := network.NewNetworkStats(cfg)

	// Process chunks sequentially
	if err := processChunks(ctx, reader, writer, outFile, transferState, stats, netStats, cfg); err != nil {
		slog.Error("Chunk processing failed", "error", err)
		protocol.SendError(writer, "Transfer failed")
		return
	}

	// Verify file hash if requested
	if cfg.VerifyHash {
		if err := verifyFileHash(ctx, reader, writer, outFile); err != nil {
			slog.Error("Hash verification failed", "error", err)
			os.Remove(outputPath)
			protocol.SendError(writer, "Hash verification failed")
			return
		}
	}

	// Cleanup and complete
	filesystem.RemoveTransferState(baseFilename, cfg.OutputDir)

	if err := protocol.SendCommand(writer, protocol.CmdComplete); err == nil {
		protocol.FlushWriter(writer)
	}

	elapsed := time.Since(stats.StartTime)
	logging.LogTransferComplete(baseFilename, fileSize, elapsed)
}

// tryResumeTransfer attempts to resume an existing transfer
func tryResumeTransfer(filename string, cfg *config.Config, fileSize, numChunks int64) (*filesystem.TransferState, bool) {
	state, err := filesystem.LoadTransferState(filename, cfg.OutputDir)
	if err != nil {
		// No existing state, start fresh
		return &filesystem.TransferState{
			Filename:       filename,
			FileSize:       fileSize,
			ChunkSize:      cfg.ChunkSize,
			NumChunks:      numChunks,
			ChunksReceived: make([]bool, numChunks),
		}, false
	}

	// Validate state compatibility
	if state.FileSize == fileSize &&
		state.ChunkSize == cfg.ChunkSize &&
		len(state.ChunksReceived) == int(numChunks) {
		slog.Info("Found compatible transfer state, resuming")
		return state, true
	}

	slog.Warn("Incompatible transfer state found, starting fresh")
	return &filesystem.TransferState{
		Filename:       filename,
		FileSize:       fileSize,
		ChunkSize:      cfg.ChunkSize,
		NumChunks:      numChunks,
		ChunksReceived: make([]bool, numChunks),
	}, false
}

// createOrOpenOutputFile creates a new file or opens existing for resume
func createOrOpenOutputFile(path string, resuming bool) (*os.File, error) {
	if resuming {
		file, err := os.OpenFile(path, os.O_RDWR, 0644)
		if err != nil {
			slog.Warn("Failed to open existing file, creating new", "error", err)
			return os.Create(path)
		}
		return file, nil
	}

	return os.Create(path)
}

// calculateResumeOffset calculates the byte offset for resume
func calculateResumeOffset(state *filesystem.TransferState) int64 {
	var receivedChunks int64
	for _, received := range state.ChunksReceived {
		if received {
			receivedChunks++
		}
	}
	return receivedChunks * state.ChunkSize
}

// processChunks handles the sequential processing of file chunks
func processChunks(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer,
	outFile *os.File, state *filesystem.TransferState, stats *progress.Stats,
	netStats *network.NetworkStats, cfg *config.Config) error {

	buffer := make([]byte, cfg.ChunkSize)

	// Save state periodically
	stateTicker := time.NewTicker(30 * time.Second)
	defer stateTicker.Stop()

	go func() {
		for {
			select {
			case <-stateTicker.C:
				if err := filesystem.SaveTransferState(state, cfg.OutputDir); err != nil {
					slog.Error("Failed to save transfer state", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Process each chunk
	for chunkIdx := int64(0); chunkIdx < state.NumChunks; chunkIdx++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip already received chunks
		if state.ChunksReceived[chunkIdx] {
			stats.UpdateTransferred(cfg.ChunkSize)
			continue
		}

		offset := chunkIdx * cfg.ChunkSize

		// Apply network delay
		if cfg.AdaptiveDelay {
			delay := netStats.GetDelay(cfg.ChunkDelay)
			time.Sleep(delay)
		} else if cfg.ChunkDelay > 0 {
			time.Sleep(cfg.ChunkDelay)
		}

		// Process chunk with retries
		actualSize, err := receiveChunkWithRetries(ctx, reader, writer, outFile,
			offset, cfg.ChunkSize, buffer, stats, cfg)
		if err != nil {
			return err
		}

		// Mark chunk as received
		state.ChunksReceived[chunkIdx] = true
		netStats.UpdateStats(actualSize)
	}

	return nil
}

// receiveChunkWithRetries receives a chunk with retry logic
func receiveChunkWithRetries(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer,
	file *os.File, offset, chunkSize int64, buffer []byte, stats *progress.Stats, cfg *config.Config) (int64, error) {

	var lastErr error

	for retry := 0; retry < cfg.Retries; retry++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		// Exponential backoff for retries
		if retry > 0 {
			backoff := time.Duration(retry*500) * time.Millisecond
			time.Sleep(backoff)
			slog.Debug("Retrying chunk", "offset", offset, "attempt", retry+1)
		}

		actualSize, err := receiveChunk(ctx, reader, writer, file, offset, chunkSize, buffer, stats, cfg)
		if err == nil {
			return actualSize, nil
		}

		lastErr = err
		slog.Warn("Chunk receive failed", "offset", offset, "retry", retry+1, "error", err)
	}

	return 0, errors.NewNetworkError("receive_chunk", "", lastErr)
}

// receiveChunk receives a single chunk from the client
func receiveChunk(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer,
	file *os.File, offset, chunkSize int64, buffer []byte, stats *progress.Stats, cfg *config.Config) (int64, error) {

	// Send chunk request
	if err := protocol.SendCommand(writer, protocol.CmdRequest); err != nil {
		return 0, err
	}

	if err := protocol.SendInt64(writer, offset); err != nil {
		return 0, err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return 0, err
	}

	// Read response
	cmdByte, err := protocol.ReadCommand(ctx, reader)
	if err != nil {
		return 0, err
	}

	if cmdByte != protocol.CmdData {
		return 0, errors.NewProtocolError("receive_chunk", "expected data command", nil)
	}

	// Read chunk size
	actualChunkSize, err := protocol.ReadInt64(ctx, reader)
	if err != nil {
		return 0, err
	}

	if actualChunkSize <= 0 || actualChunkSize > chunkSize {
		return 0, errors.NewProtocolError("receive_chunk", "invalid chunk size", nil)
	}

	// Read compression flag
	compressFlag, err := protocol.ReadCommand(ctx, reader)
	if err != nil {
		return 0, err
	}

	var data []byte

	if compressFlag == 1 {
		// Handle compressed data
		data, err = receiveCompressedChunk(ctx, reader, int(actualChunkSize))
	} else {
		// Handle uncompressed data
		data, err = receiveUncompressedChunk(ctx, reader, buffer, int(actualChunkSize))
	}

	if err != nil {
		return 0, err
	}

	// Write data to file
	if _, err := file.WriteAt(data, offset); err != nil {
		return 0, errors.NewFileSystemError("write_chunk", file.Name(), err)
	}

	stats.UpdateTransferred(actualChunkSize)
	return actualChunkSize, nil
}

// receiveCompressedChunk receives and decompresses chunk data
func receiveCompressedChunk(ctx context.Context, reader *bufio.Reader, expectedSize int) ([]byte, error) {
	// Read compressed size
	compressedSize, err := protocol.ReadInt64(ctx, reader)
	if err != nil {
		return nil, err
	}

	// Read compressed data
	compressedData := make([]byte, compressedSize)
	bytesRead := int64(0)

	for bytesRead < compressedSize {
		n, err := protocol.ReadWithContext(ctx, reader, compressedData[bytesRead:])
		if err != nil {
			return nil, err
		}
		bytesRead += int64(n)
	}

	// Decompress data
	data, err := compression.DecompressData(compressedData, expectedSize)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// receiveUncompressedChunk receives uncompressed chunk data
func receiveUncompressedChunk(ctx context.Context, reader *bufio.Reader, buffer []byte, size int) ([]byte, error) {
	bytesRead := 0

	for bytesRead < size {
		n, err := protocol.ReadWithContext(ctx, reader, buffer[bytesRead:size])
		if err != nil {
			return nil, err
		}
		bytesRead += n
	}

	return buffer[:size], nil
}

// verifyFileHash verifies the integrity of the received file
func verifyFileHash(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer, file *os.File) error {
	// Request hash from client
	if err := protocol.SendCommand(writer, protocol.CmdHash); err != nil {
		return err
	}

	if err := protocol.FlushWriter(writer); err != nil {
		return err
	}

	// Read hash response
	cmdByte, err := protocol.ReadCommand(ctx, reader)
	if err != nil {
		return err
	}

	if cmdByte != protocol.CmdHash {
		return errors.NewProtocolError("verify_hash", "expected hash command", nil)
	}

	sourceHash, err := protocol.ReadString(ctx, reader)
	if err != nil {
		return err
	}

	// Calculate hash of received file
	receivedHash, err := filesystem.CalculateFileHash(file)
	if err != nil {
		return err
	}

	// Compare hashes
	if sourceHash != receivedHash {
		return errors.NewValidationError("hash", receivedHash, "hash mismatch with source")
	}

	slog.Info("File hash verified successfully", "hash_algorithm", "MD5")
	return nil
}
