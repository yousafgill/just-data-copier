/*
Copyright 2025 Yousaf Gill. All rights reserved.
Use of this source code is governed by the MIT license
that can be found in the LICENSE file.

JustDataCopier is a high-performance network file transfer utility designed
to efficiently transfer large files across network connections with features
like parallel chunk transfers, adaptive network handling, compression, and
transfer resume capabilities.

The program operates in two modes:

 1. Server Mode: Receives files from clients and verifies their integrity

 2. Client Mode: Sends files to a server with configurable optimizations

    Author: Yousaf Gill <yousafgill@gmail.com>
    Repository: https://github.com/yousafgill/just-data-copier
    Detail: provided in README.md
*/
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"justdatacopier/internal/client"
	"justdatacopier/internal/config"
	"justdatacopier/internal/logging"
	"justdatacopier/internal/server"
)

func main() {
	// Setup structured logging first
	if err := logging.SetupLogger(); err != nil {
		slog.Error("Failed to setup logging", "error", err)
		os.Exit(1)
	}

	// Parse command line arguments
	cfg, err := config.ParseFlags()
	if err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	// Log configuration
	logging.LogConfig(cfg)

	// Set GOMAXPROCS based on worker configuration
	runtime.GOMAXPROCS(cfg.Workers)
	slog.Info("Runtime configured", "gomaxprocs", cfg.Workers)

	// Set up signal handling for graceful shutdown
	setupSignalHandling()

	// Run in appropriate mode
	if cfg.IsServer {
		if err := server.Run(cfg); err != nil {
			logging.LogError(err, "server")
			os.Exit(1)
		}
	} else {
		if err := client.Run(cfg); err != nil {
			logging.LogError(err, "client")
			os.Exit(1)
		}
	}
}

// setupSignalHandling sets up handlers for OS signals to ensure clean shutdown
func setupSignalHandling() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		slog.Info("Received shutdown signal", "signal", sig)

		// Allow some time for cleanup
		time.Sleep(500 * time.Millisecond)

		slog.Info("Application shutting down gracefully")
		os.Exit(0)
	}()
}

// Configuration parameters
type Config struct {
	// Server mode
	IsServer      bool
	ListenAddress string
	OutputDir     string

	// Client mode
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
	Retries       int           // Number of retry attempts for failed operations
	ChunkDelay    time.Duration // Delay between chunk transfers to prevent network overload
	AdaptiveDelay bool          // Use adaptive delay based on network conditions
	MinDelay      time.Duration // Minimum delay for adaptive networking
	MaxDelay      time.Duration // Maximum delay for adaptive networking
}

// Statistics for monitoring transfer
type Stats struct {
	TotalBytes       int64
	TransferredBytes atomic.Int64
	StartTime        time.Time
	FileSize         int64
}

// NetworkStats tracks network performance metrics to enable adaptive chunk delays
type NetworkStats struct {
	LastChunkTime   time.Time
	LastChunkSize   int64
	AvgTransferRate float64 // bytes per second
	DelayMultiplier float64 // adjusts delay up/down
	minDelay        time.Duration
	maxDelay        time.Duration
}

// NetworkProfile contains information about the network environment
type NetworkProfile struct {
	RTT              time.Duration // Round-trip time
	Bandwidth        int64         // Estimated bandwidth in bytes/second
	PacketLoss       float64       // Estimated packet loss rate
	OptimalChunkSize int64         // Calculated optimal chunk size
}

// NewNetworkStats initializes a new NetworkStats instance with values from config
func NewNetworkStats(config Config) *NetworkStats {
	minDelay := time.Millisecond
	maxDelay := 100 * time.Millisecond

	// Use config values if adaptive delay is enabled
	if config.AdaptiveDelay {
		minDelay = config.MinDelay
		maxDelay = config.MaxDelay
	}

	return &NetworkStats{
		LastChunkTime:   time.Now(),
		DelayMultiplier: 1.0,
		minDelay:        minDelay,
		maxDelay:        maxDelay,
	}
}

// UpdateStats updates network statistics based on the latest chunk transfer
func (ns *NetworkStats) UpdateStats(chunkSize int64) {
	now := time.Now()
	duration := now.Sub(ns.LastChunkTime)

	// Calculate bytes per second
	if duration > 0 {
		currentRate := float64(chunkSize) / duration.Seconds()
		// Save previous values for comparison
		prevMultiplier := ns.DelayMultiplier

		// Smooth the rate with exponential moving average
		if ns.AvgTransferRate == 0 {
			ns.AvgTransferRate = currentRate
		} else {
			ns.AvgTransferRate = 0.7*ns.AvgTransferRate + 0.3*currentRate
		}

		// Adjust delay multiplier based on transfer rate
		// If rate is decreasing, increase delay
		if currentRate < 0.7*ns.AvgTransferRate {
			ns.DelayMultiplier *= 1.2
		} else if currentRate > 1.2*ns.AvgTransferRate {
			// If rate is increasing, decrease delay
			ns.DelayMultiplier *= 0.8
		}

		// Keep multiplier in reasonable bounds
		if ns.DelayMultiplier < 0.1 {
			ns.DelayMultiplier = 0.1
		} else if ns.DelayMultiplier > 10 {
			ns.DelayMultiplier = 10
		}

		// Log significant changes in network conditions
		if ns.DelayMultiplier != prevMultiplier {
			// Convert bytes/sec to MB/sec for readability
			currentRateMB := currentRate / (1024 * 1024)
			avgRateMB := ns.AvgTransferRate / (1024 * 1024)

			if ns.DelayMultiplier > prevMultiplier {
				infoLog.Printf("Network congestion detected - Rate: %.2f MB/s (Avg: %.2f MB/s) - Increasing delay factor to %.1f",
					currentRateMB, avgRateMB, ns.DelayMultiplier)
			} else {
				infoLog.Printf("Network improving - Rate: %.2f MB/s (Avg: %.2f MB/s) - Decreasing delay factor to %.1f",
					currentRateMB, avgRateMB, ns.DelayMultiplier)
			}
		}
	}

	ns.LastChunkTime = now
	ns.LastChunkSize = chunkSize
}

// GetDelay calculates the adaptive delay based on current network conditions
func (ns *NetworkStats) GetDelay(baseDelay time.Duration) time.Duration {
	delay := time.Duration(float64(baseDelay) * ns.DelayMultiplier)

	// Apply bounds
	if delay < ns.minDelay {
		delay = ns.minDelay
	}
	if delay > ns.maxDelay {
		delay = ns.maxDelay
	}

	// Log delay information periodically (every ~50 calls - using a simple counter would require a mutex)
	// Use time based threshold to avoid too frequent logging
	if time.Since(ns.LastChunkTime) > 10*time.Second && ns.LastChunkTime.Unix() > 0 {
		infoLog.Printf("Currently using adaptive delay: %v (multiplier: %.1f)",
			delay, ns.DelayMultiplier)
	}

	return delay
}

type TransferState struct {
	Filename       string
	FileSize       int64
	ChunkSize      int64
	NumChunks      int64
	ChunksReceived []bool
	LastModified   time.Time
}

func saveTransferState(state TransferState, outputDir string) error {
	stateFile := filepath.Join(outputDir, state.Filename+".justdatacopier.state")
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0644)
}

func loadTransferState(filename string, outputDir string) (TransferState, error) {
	stateFile := filepath.Join(outputDir, filename+".justdatacopier.state")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return TransferState{}, err
	}

	var state TransferState
	if err := json.Unmarshal(data, &state); err != nil {
		return TransferState{}, err
	}
	return state, nil
}

// Command operation codes
const (
	CMD_INIT     = 1 // Initialize transfer
	CMD_REQUEST  = 2 // Request chunk
	CMD_DATA     = 3 // Send chunk data
	CMD_COMPLETE = 4 // Transfer complete
	CMD_ERROR    = 5 // Error occurred
	CMD_HASH     = 6 // File hash for verification
	CMD_PING     = 7 // Ping for network profiling
	CMD_PONG     = 8 // Pong response to ping
)

// Initialize logging
var (
	infoLog  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	errorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func main() {
	// Parse command line arguments
	config := parseFlags()

	// Set GOMAXPROCS based on worker configuration
	runtime.GOMAXPROCS(config.Workers)

	// Print configuration details
	printConfigInfo(config)

	// Setup logging
	setupLogging()

	// Set up signal handling for graceful shutdown
	setupSignalHandling()

	if config.IsServer {
		runServer(config)
	} else {
		runClient(config)
	}
}

// setupSignalHandling sets up handlers for OS signals to ensure clean shutdown
func setupSignalHandling() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		infoLog.Printf("Received signal %v, shutting down gracefully...", sig)

		// Allow some time for state files to be written
		time.Sleep(500 * time.Millisecond)

		os.Exit(0)
	}()
}

func setupLogging() {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		errorLog.Printf("Failed to create logs directory: %v", err)
		// Continue anyway, will log to console
	}

	// Create log file with timestamp
	logFileName := fmt.Sprintf("logs/justdatacopier_%s.log",
		time.Now().Format("20060102_150405"))

	logFile, err := os.Create(logFileName)
	if err != nil {
		errorLog.Printf("Failed to create log file: %v", err)
		// Continue with console logging
		return
	}

	// Create multi-writer to log to both console and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	infoLog = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime)

	errorMultiWriter := io.MultiWriter(os.Stderr, logFile)
	errorLog = log.New(errorMultiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Helper functions for safe I/O operations with context

// readWithContext reads data from a reader with context cancellation support
func readWithContext(ctx context.Context, reader *bufio.Reader, buffer []byte) (int, error) {
	type readResult struct {
		n   int
		err error
	}

	// Use buffered channel to ensure the goroutine doesn't block if we exit early
	resultCh := make(chan readResult, 1)

	// Create a goroutine-local context that we can cancel if needed
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel() // Always cancel the goroutine-local context when we return

	go func() {
		// Use a select to check if the context is done before proceeding with read
		select {
		case <-readCtx.Done():
			// Parent function has moved on, clean up and exit
			return
		default:
			// Proceed with the read
			n, err := reader.Read(buffer)

			// Only send on channel if context is not cancelled
			select {
			case <-readCtx.Done():
				// Parent function has moved on, don't send result
				return
			default:
				resultCh <- readResult{n, err}
			}
		}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		// Cancel the goroutine to prevent leaks
		cancel()
		return 0, ctx.Err()
	}
}

// readByteWithContext reads a single byte with context cancellation support
func readByteWithContext(ctx context.Context, reader *bufio.Reader) (byte, error) {
	type byteResult struct {
		b   byte
		err error
	}

	resultCh := make(chan byteResult, 1)
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-readCtx.Done():
			return
		default:
			b, err := reader.ReadByte()
			select {
			case <-readCtx.Done():
				return
			default:
				resultCh <- byteResult{b, err}
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.b, result.err
	case <-ctx.Done():
		cancel()
		return 0, ctx.Err()
	}
}

// readStringWithContext reads a string until delimiter with context cancellation support
func readStringWithContext(ctx context.Context, reader *bufio.Reader, delim byte) (string, error) {
	type stringResult struct {
		s   string
		err error
	}

	resultCh := make(chan stringResult, 1)
	readCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-readCtx.Done():
			return
		default:
			str, err := reader.ReadString(delim)
			select {
			case <-readCtx.Done():
				return
			default:
				resultCh <- stringResult{str, err}
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.s, result.err
	case <-ctx.Done():
		cancel()
		return "", ctx.Err()
	}
}

func printConfigInfo(config Config) {
	fmt.Println("------------------------------------------------------------")
	fmt.Println("JustDataCopier  : High Performance Network File Transfer Utility")
	fmt.Println("Author          : Yousaf Gill ")
	fmt.Println("copyright       : 2025 Freeware. All rights reserved.")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("Mode: %s\n", iif(config.IsServer, "Server", "Client"))

	if config.IsServer {
		fmt.Printf("Listen address: %s\n", config.ListenAddress)
		fmt.Printf("Output directory: %s\n", config.OutputDir)
	} else {
		fmt.Printf("Server address: %s\n", config.ServerAddress)
		fmt.Printf("Source file: %s\n", config.FilePath)
	}

	fmt.Printf("Workers: %d\n", config.Workers)
	fmt.Printf("Chunk size: %d bytes (%.2f MB)\n", config.ChunkSize, float64(config.ChunkSize)/1024/1024)
	fmt.Printf("Buffer size: %d bytes (%.2f MB)\n", config.BufferSize, float64(config.BufferSize)/1024/1024)
	fmt.Printf("Compression: %v\n", config.Compression)
	fmt.Printf("Verify file: %v\n", config.VerifyHash)
	fmt.Printf("Show progress: %v\n", config.ShowProgress)
	fmt.Printf("Error handling: Context-based cancellation\n")
	fmt.Printf("Adaptive delay: %v\n", config.AdaptiveDelay)
	if config.AdaptiveDelay {
		fmt.Printf("Delay range: %v - %v\n", config.MinDelay, config.MaxDelay)
	} else {
		fmt.Printf("Fixed delay: %v\n", config.ChunkDelay)
	}
	fmt.Println("------------------------------------------------")
}

// Helper function for ternary conditional operation
func iif(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func parseFlags() Config {
	// Server flags
	isServer := flag.Bool("server", false, "Run in server mode")
	listenAddr := flag.String("listen", "0.0.0.0:8000", "Address to listen on (server mode)")
	outputDir := flag.String("output", "./output", "Directory to store received files (server mode)")

	// Client flags
	serverAddr := flag.String("connect", "localhost:8000", "Server address to connect to (client mode)")
	filePath := flag.String("file", "", "File to transfer (client mode)")

	// Common flags
	chunkSize := flag.Int64("chunk", 2*1024*1024, "Chunk size in bytes (2MB default, smaller chunks for more reliable transfer)")
	bufferSize := flag.Int("buffer", 512*1024, "Buffer size in bytes (512KB default for better network handling)")
	workers := flag.Int("workers", runtime.NumCPU()/2, "Number of worker threads (default: half of CPU cores)")
	compression := flag.Bool("compress", false, "Enable gzip compression to reduce network transfer size (improves performance for compressible data)")
	verifyHash := flag.Bool("verify", true, "Verify file integrity with MD5")
	showProgress := flag.Bool("progress", true, "Show progress during transfer")
	timeout := flag.Duration("timeout", 2*time.Minute, "Operation timeout for individual operations")
	retries := flag.Int("retries", 5, "Number of retries for failed operations")
	chunkDelay := flag.Duration("delay", 10*time.Millisecond, "Delay between chunk transfers in milliseconds")
	adaptiveDelay := flag.Bool("adaptive", false, "Use adaptive delay based on network conditions")
	minDelay := flag.Duration("min-delay", 1*time.Millisecond, "Minimum delay for adaptive networking")
	maxDelay := flag.Duration("max-delay", 100*time.Millisecond, "Maximum delay for adaptive networking")

	flag.Parse()

	return Config{
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
}

// SERVER IMPLEMENTATION

func runServer(config Config) {
	infoLog.Printf("Starting server on %s", config.ListenAddress)
	infoLog.Printf("Using %d worker threads", config.Workers)
	infoLog.Printf("Output directory: %s", config.OutputDir)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		errorLog.Fatalf("Failed to create output directory: %v", err)
	}

	// Start listener
	listener, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		errorLog.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	infoLog.Println("Server ready to accept connections")

	// Accept and handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			errorLog.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config Config) {
	defer conn.Close()
	infoLog.Printf("New connection from %s", conn.RemoteAddr())

	// Set connection deadline to zero (no timeout) for persistent connections
	if err := conn.SetDeadline(time.Time{}); err != nil {
		errorLog.Printf("Failed to disable connection deadline: %v", err)
		return
	}

	// Apply TCP-specific optimizations if possible
	if tcpConn, isTCP := conn.(*net.TCPConn); isTCP {
		// Enable keep-alive to detect dead connections
		if err := tcpConn.SetKeepAlive(true); err != nil {
			errorLog.Printf("Warning: Failed to enable TCP keepalives: %v", err)
		}

		// Set keep-alive interval
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			errorLog.Printf("Warning: Failed to set TCP keepalive period: %v", err)
		}

		// Disable Nagle's algorithm for better performance with our own chunking
		if err := tcpConn.SetNoDelay(true); err != nil {
			errorLog.Printf("Warning: Failed to disable Nagle's algorithm: %v", err)
		}

		// Set larger buffer sizes for high throughput
		if err := tcpConn.SetReadBuffer(1024 * 1024); err != nil {
			errorLog.Printf("Warning: Failed to set TCP read buffer: %v", err)
		}

		if err := tcpConn.SetWriteBuffer(1024 * 1024); err != nil {
			errorLog.Printf("Warning: Failed to set TCP write buffer: %v", err)
		}
	}

	// Create buffered reader and writer with optimized buffer size
	reader := bufio.NewReaderSize(conn, config.BufferSize)
	writer := bufio.NewWriterSize(conn, config.BufferSize)

	// Loop to handle multiple commands on the same connection
	for {
		// Read command
		cmdByte, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				// Client closed connection, this is normal
				infoLog.Printf("Connection closed by client: %s", conn.RemoteAddr())
			} else {
				errorLog.Printf("Failed to read command: %v", err)
			}
			return
		}

		switch cmdByte {
		case CMD_INIT:
			handleFileTransfer(reader, writer, conn, config)
			return // After file transfer is complete, close the connection
		case CMD_PING: // Handle ping requests for network profiling
			// Respond to ping with pong
			if err := writer.WriteByte(CMD_PONG); err != nil {
				errorLog.Printf("Failed to send pong command: %v", err)
				return
			}

			if err := writer.Flush(); err != nil {
				errorLog.Printf("Failed to flush pong response: %v", err)
				return
			}
			// Continue the loop to handle more commands
		default:
			errorLog.Printf("Unknown command: %d", cmdByte)
			sendError(writer, "Unknown command")
			return
		}
	}
}

// --------------- start of handleFileTransfer ------------------//
func handleFileTransfer(reader *bufio.Reader, writer *bufio.Writer, conn net.Conn, config Config) {
	// Read filename
	filename, err := reader.ReadString('\n')
	if err != nil {
		errorLog.Printf("Failed to read filename: %v", err)
		sendError(writer, "Failed to read filename")
		return
	}
	filename = filename[:len(filename)-1] // Remove newline
	baseFilename := filepath.Base(filename)

	// Read file size
	fileSizeStr, err := reader.ReadString('\n')
	if err != nil {
		errorLog.Printf("Failed to read file size: %v", err)
		sendError(writer, "Failed to read file size")
		return
	}
	fileSize, err := strconv.ParseInt(fileSizeStr[:len(fileSizeStr)-1], 10, 64)
	if err != nil {
		errorLog.Printf("Failed to parse file size: %v", err)
		sendError(writer, "Invalid file size")
		return
	}

	// Create output path
	outputPath := filepath.Join(config.OutputDir, baseFilename)
	var outFile *os.File

	// Initialize transfer state variables
	var resuming bool = false
	numChunks := (fileSize + config.ChunkSize - 1) / config.ChunkSize
	chunkStatus := make([]bool, numChunks)
	var resumeOffset int64 = 0

	// Try to load existing transfer state
	stateObj, stateErr := loadTransferState(baseFilename, config.OutputDir)
	if stateErr == nil && stateObj.FileSize == fileSize && stateObj.ChunkSize == config.ChunkSize && len(stateObj.ChunksReceived) == int(numChunks) {
		infoLog.Printf("Resuming partial transfer for %s", baseFilename)
		resuming = true
		chunkStatus = stateObj.ChunksReceived

		// Calculate resume progress
		receivedChunks := 0
		for _, received := range chunkStatus {
			if received {
				receivedChunks++
			}
		}
		resumeOffset = int64(receivedChunks) * config.ChunkSize
		//infoLog.Printf("Resuming from %.2f MB (%d%%)", float64(resumeOffset)/1024/1024, int(float64(resumeOffset)/int(float64(fileSize)*100))

	}

	// Open/Create output file
	if resuming {
		outFile, err = os.OpenFile(outputPath, os.O_RDWR, 0644)
		if err != nil {
			errorLog.Printf("Failed to open existing file: %v", err)
			resuming = false
		}
	}
	if !resuming {
		outFile, err = os.Create(outputPath)
		if err != nil {
			errorLog.Printf("Failed to create file: %v", err)
			sendError(writer, "File creation failed")
			return
		}
		preallocateFile(outFile, fileSize) // Best effort
	}
	defer outFile.Close()

	infoLog.Printf("Receiving %s (%.2f MB)", baseFilename, float64(fileSize)/1024/1024)

	// Initialize tracking
	stats := Stats{
		TotalBytes: fileSize,
		StartTime:  time.Now(),
		FileSize:   fileSize,
	}
	stats.TransferredBytes.Store(resumeOffset)

	// Setup context and progress
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var progressDone chan struct{}
	if config.ShowProgress {
		progressDone = make(chan struct{})
		go reportProgress(&stats, progressDone)
		defer close(progressDone)
	}

	// Setup network stats
	netStats := NewNetworkStats(config)
	if !config.AdaptiveDelay {
		infoLog.Printf("Using fixed delay: %v", config.ChunkDelay)
	}

	// State saving ticker
	stateTicker := time.NewTicker(30 * time.Second)
	defer stateTicker.Stop()
	go func() {
		for {
			select {
			case <-stateTicker.C:
				currentState := TransferState{
					Filename:       baseFilename,
					FileSize:       fileSize,
					ChunkSize:      config.ChunkSize,
					NumChunks:      numChunks,
					ChunksReceived: append([]bool{}, chunkStatus...),
					LastModified:   time.Now(),
				}
				if err := saveTransferState(currentState, config.OutputDir); err != nil {
					errorLog.Printf("State save failed: %v", err)
				}
			case <-ctx.Done():
				return
			case <-progressDone:
				return
			}
		}
	}()

	// Sequential chunk processing
	buffer := make([]byte, config.ChunkSize)
	for chunkIdx := int64(0); chunkIdx < numChunks; chunkIdx++ {
		if ctx.Err() != nil {
			break // Context cancelled
		}

		offset := chunkIdx * config.ChunkSize
		if chunkStatus[chunkIdx] {
			stats.TransferredBytes.Add(config.ChunkSize)
			continue // Skip already received chunks
		}

		// Apply network delay
		if config.AdaptiveDelay {
			delay := netStats.GetDelay(config.ChunkDelay)
			time.Sleep(delay)
		} else if config.ChunkDelay > 0 {
			time.Sleep(config.ChunkDelay)
		}

		// Process chunk with retries
		var lastErr error
		for retry := 0; retry < config.Retries; retry++ {
			err := receiveChunkWithContext(ctx, reader, writer, outFile, offset,
				config.ChunkSize, buffer, &stats, config)
			if err == nil {
				chunkStatus[chunkIdx] = true
				actualSize := config.ChunkSize
				if offset+config.ChunkSize > fileSize {
					actualSize = fileSize - offset
				}
				netStats.UpdateStats(actualSize)
				break
			}
			lastErr = err
			errorLog.Printf("Chunk %d failed (retry %d/%d): %v",
				chunkIdx, retry+1, config.Retries, err)
			time.Sleep(time.Duration(retry+1) * 500 * time.Millisecond)
		}

		if lastErr != nil {
			errorLog.Printf("Permanent failure on chunk %d: %v", chunkIdx, lastErr)
			sendError(writer, fmt.Sprintf("Chunk %d failed: %v", chunkIdx, lastErr))
			cancel()
			return
		}
	}

	// Final verification
	if config.VerifyHash {
		if err := verifyFileHash(reader, writer, outFile); err != nil {
			errorLog.Printf("Verification failed: %v", err)
			os.Remove(outputPath)
			sendError(writer, "Hash verification failed")
			return
		}
	}

	// Cleanup and completion
	statePath := filepath.Join(config.OutputDir, baseFilename+".justdatacopier.state")
	os.Remove(statePath) // Best effort

	if err := writer.WriteByte(CMD_COMPLETE); err == nil {
		writer.Flush()
	}

	elapsed := time.Since(stats.StartTime)
	infoLog.Printf("Transfer complete: %.2f MB in %v (%.2f MB/s)",
		float64(fileSize)/1024/1024,
		elapsed.Round(time.Second),
		float64(fileSize)/1024/1024/elapsed.Seconds())
}

// --------------- end of handleFileTransfer ------------------//

func receiveChunkWithContext(ctx context.Context, reader *bufio.Reader, writer *bufio.Writer, file *os.File, offset int64, chunkSize int64, buffer []byte, stats *Stats, config Config) error {
	// Use the configured retry count
	maxRetries := config.Retries
	if maxRetries <= 0 {
		maxRetries = 5 // Default if not properly set
	}

	var lastErr error

	for retry := 0; retry < maxRetries && ctx.Err() == nil; retry++ {
		// If retrying, wait a bit with exponential backoff
		if retry > 0 {
			backoff := time.Duration(retry*500) * time.Millisecond
			select {
			case <-time.After(backoff):
				// Continue after backoff
			case <-ctx.Done():
				return fmt.Errorf("operation cancelled during backoff: %w", ctx.Err())
			}
			infoLog.Printf("Retrying chunk at offset %d (attempt %d/%d)", offset, retry+1, maxRetries)
		}

		// Send chunk request
		if err := writer.WriteByte(CMD_REQUEST); err != nil {
			lastErr = fmt.Errorf("failed to send chunk request command: %w", err)
			continue
		}

		// Send chunk offset (ensure it ends with a newline)
		if _, err := fmt.Fprintf(writer, "%d\n", offset); err != nil {
			lastErr = fmt.Errorf("failed to send chunk offset: %w", err)
			continue
		}

		if err := writer.Flush(); err != nil {
			lastErr = fmt.Errorf("failed to flush request: %w", err)
			continue
		}

		// Read response command with context timeout
		cmdByte, cmdErr := readByteWithContext(ctx, reader)
		if cmdErr != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("context deadline exceeded waiting for command: %w", ctx.Err())
			}
			lastErr = fmt.Errorf("failed to read response command: %w", cmdErr)
			continue
		}

		if cmdByte != CMD_DATA {
			lastErr = fmt.Errorf("expected data command, got %d", cmdByte)
			continue
		}

		// Read chunk size with context timeout
		chunkSizeStr, sizeErr := readStringWithContext(ctx, reader, '\n')
		if sizeErr != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("context deadline exceeded reading chunk size: %w", ctx.Err())
			}
			lastErr = fmt.Errorf("failed to read chunk size: %w", sizeErr)
			continue
		}

		// Ensure we have valid string data before parsing
		chunkSizeStr = strings.TrimSpace(chunkSizeStr)
		if len(chunkSizeStr) == 0 {
			lastErr = fmt.Errorf("received empty chunk size")
			continue
		}

		actualChunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
		if err != nil {
			// Log the actual string received for debugging
			lastErr = fmt.Errorf("invalid chunk size '%s': %w", chunkSizeStr, err)
			continue
		}

		// Check if chunk size is valid
		if actualChunkSize <= 0 || actualChunkSize > chunkSize {
			lastErr = fmt.Errorf("invalid chunk size: %d (max: %d)", actualChunkSize, chunkSize)
			continue
		}

		// Read compression flag
		compressFlag, err := readByteWithContext(ctx, reader)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("context deadline exceeded reading compression flag: %w", ctx.Err())
			}
			lastErr = fmt.Errorf("failed to read compression flag: %w", err)
			continue
		}

		// Handle based on compression flag
		if compressFlag == 1 {
			// Compressed data

			// Read compressed size
			compSizeStr, err := readStringWithContext(ctx, reader, '\n')
			if err != nil {
				if ctx.Err() != nil {
					return fmt.Errorf("context deadline exceeded reading compressed size: %w", ctx.Err())
				}
				lastErr = fmt.Errorf("failed to read compressed size: %w", err)
				continue
			}

			// Parse compressed size
			compSizeStr = strings.TrimSpace(compSizeStr)
			compressedSize, err := strconv.ParseInt(compSizeStr, 10, 64)
			if err != nil {
				lastErr = fmt.Errorf("invalid compressed size '%s': %w", compSizeStr, err)
				continue
			}

			// Read compressed data
			compressedBuffer := make([]byte, compressedSize)
			bytesRead := int64(0)
			var readSuccess bool = true

			// Read the compressed data
			for bytesRead < compressedSize && readSuccess {
				n, err := readWithContext(ctx, reader, compressedBuffer[bytesRead:compressedSize])
				if err != nil {
					if ctx.Err() != nil {
						return fmt.Errorf("context deadline exceeded reading compressed data: %w", ctx.Err())
					}
					lastErr = fmt.Errorf("failed to read compressed data after %d/%d bytes: %w",
						bytesRead, compressedSize, err)
					readSuccess = false
					break
				}
				bytesRead += int64(n)
			}

			// If we had an error reading, continue to the next retry
			if !readSuccess {
				continue
			}

			// If we didn't read the full compressed chunk, retry
			if bytesRead < compressedSize {
				lastErr = fmt.Errorf("incomplete compressed chunk read: %d/%d bytes", bytesRead, compressedSize)
				continue
			}

			// Create a reader for the compressed data
			compReader := bytes.NewReader(compressedBuffer)

			// Create a gzip reader
			gzipReader, err := gzip.NewReader(compReader)
			if err != nil {
				lastErr = fmt.Errorf("failed to create gzip reader: %w", err)
				continue
			}

			// Read decompressed data
			decompressedSize, err := io.ReadFull(gzipReader, buffer)
			if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
				gzipReader.Close()
				lastErr = fmt.Errorf("failed to decompress data: %w", err)
				continue
			}

			// Close the gzip reader
			if err := gzipReader.Close(); err != nil {
				lastErr = fmt.Errorf("error closing gzip reader: %w", err)
				continue
			}

			// Log compression information
			if retry == 0 { // Only log on first try to avoid spam
				compressionRatio := float64(decompressedSize) / float64(compressedSize)
				infoLog.Printf("Chunk decompressed: %d → %d bytes (%.2fx ratio)",
					compressedSize, decompressedSize, compressionRatio)
			}

			// Write decompressed data to file
			if _, err := file.WriteAt(buffer[:decompressedSize], offset); err != nil {
				lastErr = fmt.Errorf("failed to write decompressed chunk to file: %w", err)
				continue
			}

			// Update stats with original chunk size for accurate progress reporting
			stats.TransferredBytes.Add(actualChunkSize)

		} else {
			// Uncompressed data
			// Read chunk data with context timeout
			bytesRead := int64(0)
			var readSuccess bool = true

			// Read the chunk data
			for bytesRead < actualChunkSize && readSuccess {
				// Use our dedicated readWithContext function that properly handles cancellation
				n, err := readWithContext(ctx, reader, buffer[bytesRead:actualChunkSize])
				if err != nil {
					if ctx.Err() != nil {
						return fmt.Errorf("context deadline exceeded reading chunk data: %w", ctx.Err())
					}
					lastErr = fmt.Errorf("failed to read chunk data after %d/%d bytes: %w",
						bytesRead, actualChunkSize, err)
					readSuccess = false
					break
				}
				bytesRead += int64(n)
			}

			// If we had an error reading, continue to the next retry
			if !readSuccess {
				continue
			}

			// If we didn't read the full chunk, retry
			if bytesRead < actualChunkSize {
				lastErr = fmt.Errorf("incomplete chunk read: %d/%d bytes", bytesRead, actualChunkSize)
				continue
			}

			// Write chunk to file
			if _, err := file.WriteAt(buffer[:actualChunkSize], offset); err != nil {
				lastErr = fmt.Errorf("failed to write chunk to file: %w", err)
				continue
			}

			// Update stats
			stats.TransferredBytes.Add(actualChunkSize)
		}

		// Success, return nil
		return nil
	}

	// Check if context was cancelled
	if ctx.Err() != nil {
		return fmt.Errorf("operation cancelled: %w", ctx.Err())
	}

	// If we get here, all retries failed
	return fmt.Errorf("all retries failed for chunk at offset %d: %w", offset, lastErr)
}

func verifyFileHash(reader *bufio.Reader, writer *bufio.Writer, file *os.File) error {
	// Create a context with timeout for hash verification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Generous timeout for large files
	defer cancel()

	// Set up error channel for goroutine communication
	errCh := make(chan error, 1)

	// Perform hash verification in a goroutine to handle timeouts
	go func() {
		// Request hash
		if err := writer.WriteByte(CMD_HASH); err != nil {
			errCh <- fmt.Errorf("failed to send hash request: %w", err)
			return
		}
		if err := writer.Flush(); err != nil {
			errCh <- fmt.Errorf("failed to flush hash request: %w", err)
			return
		}

		// Read source hash with timeout using our context functions
		cmdByte, err := readByteWithContext(ctx, reader)
		if err != nil {
			errCh <- fmt.Errorf("failed to read hash response: %w", err)
			return
		}
		if cmdByte != CMD_HASH {
			errCh <- fmt.Errorf("expected hash command, got %d", cmdByte)
			return
		}

		sourceHashStr, err := readStringWithContext(ctx, reader, '\n')
		if err != nil {
			errCh <- fmt.Errorf("failed to read source hash: %w", err)
			return
		}
		sourceHash := sourceHashStr[:len(sourceHashStr)-1] // Remove newline

		// Calculate hash of received file
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			errCh <- fmt.Errorf("failed to seek file: %w", err)
			return
		}

		// Use a buffered hash calculation to prevent blocking
		hash := md5.New()
		buffer := make([]byte, 4*1024*1024) // 4MB buffer for hashing

		// Copy with buffer to calculate hash
		remaining := true
		for remaining && ctx.Err() == nil {
			select {
			case <-ctx.Done():
				errCh <- fmt.Errorf("hash calculation timed out: %w", ctx.Err())
				return
			default:
				nr, err := file.Read(buffer)
				if err != nil {
					if err == io.EOF {
						remaining = false
					} else {
						errCh <- fmt.Errorf("failed to read file during hash calculation: %w", err)
						return
					}
				}

				if nr > 0 {
					_, err := hash.Write(buffer[:nr])
					if err != nil {
						errCh <- fmt.Errorf("failed to update hash: %w", err)
						return
					}
				}
			}
		}

		receivedHash := hex.EncodeToString(hash.Sum(nil))

		// Compare hashes
		if sourceHash != receivedHash {
			errCh <- fmt.Errorf("hash mismatch: %s != %s", sourceHash, receivedHash)
			return
		}

		// Success - send nil error to channel
		errCh <- nil
	}()

	// Wait for result or context cancellation
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("hash verification timed out: %w", ctx.Err())
	}
}

func sendError(writer *bufio.Writer, message string) {
	_ = writer.WriteByte(CMD_ERROR)
	_, _ = writer.WriteString(message + "\n")
	_ = writer.Flush()
}

// CLIENT IMPLEMENTATION

func runClient(config Config) {
	if config.FilePath == "" {
		errorLog.Fatal("File path is required in client mode")
	}

	infoLog.Printf("Starting client, connecting to %s", config.ServerAddress)
	infoLog.Printf("Using %d worker threads", config.Workers)

	// Check if file exists
	file, err := os.Open(config.FilePath)
	if err != nil {
		errorLog.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		errorLog.Fatalf("Failed to get file info: %v", err)
	}

	if fileInfo.IsDir() {
		errorLog.Fatal("Cannot transfer directories")
	}

	// Connect to server
	conn, err := net.Dial("tcp", config.ServerAddress)
	if err != nil {
		errorLog.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Set connection deadline to zero (no timeout) for persistent connections
	if err := conn.SetDeadline(time.Time{}); err != nil {
		errorLog.Fatalf("Failed to disable connection deadline: %v", err)
	}

	// Create a TCP-specific connection to access more TCP-level features
	tcpConn, isTCP := conn.(*net.TCPConn)
	if isTCP {
		// Enable keep-alive to detect dead connections
		if err := tcpConn.SetKeepAlive(true); err != nil {
			errorLog.Printf("Warning: Failed to enable TCP keepalives: %v", err)
		}

		// Set keep-alive interval
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			errorLog.Printf("Warning: Failed to set TCP keepalive period: %v", err)
		}

		// Disable Nagle's algorithm for better performance with our own chunking
		if err := tcpConn.SetNoDelay(true); err != nil {
			errorLog.Printf("Warning: Failed to disable Nagle's algorithm: %v", err)
		}

		// Set larger buffer sizes for high throughput
		if err := tcpConn.SetReadBuffer(1024 * 1024); err != nil {
			errorLog.Printf("Warning: Failed to set TCP read buffer: %v", err)
		}

		if err := tcpConn.SetWriteBuffer(1024 * 1024); err != nil {
			errorLog.Printf("Warning: Failed to set TCP write buffer: %v", err)
		}
	} else {
		infoLog.Printf("Connection is not TCP, skipping TCP-specific optimizations")
	}

	// Create buffered reader and writer with appropriate size
	reader := bufio.NewReaderSize(conn, config.BufferSize)
	writer := bufio.NewWriterSize(conn, config.BufferSize)

	// Perform network profiling
	infoLog.Printf("Performing network profiling to optimize transfer settings...")
	profile := profileNetwork(conn)
	infoLog.Printf("Network profile: RTT=%v, Optimal chunk size=%d bytes (%.2f MB)",
		profile.RTT, profile.OptimalChunkSize, float64(profile.OptimalChunkSize)/1024/1024)

	// Ensure the connection deadline is reset and buffers are flushed
	if err := conn.SetDeadline(time.Time{}); err != nil {
		errorLog.Printf("Warning: Failed to reset connection deadline after profiling: %v", err)
	}

	// Re-create reader and writer with optimal buffer sizes based on profile
	optimalBufferSize := max(config.BufferSize, int(profile.OptimalChunkSize/4))
	reader = bufio.NewReaderSize(conn, optimalBufferSize)
	writer = bufio.NewWriterSize(conn, optimalBufferSize)

	infoLog.Printf("Using optimized buffer size: %d bytes (%.2f MB)",
		optimalBufferSize, float64(optimalBufferSize)/1024/1024)

	// Adjust config based on profile
	originalWorkers := config.Workers
	originalChunkSize := config.ChunkSize

	if profile.RTT > 100*time.Millisecond {
		// High latency network, reduce workers and increase chunk size
		config.Workers = max(1, config.Workers/2)
		config.ChunkSize = profile.OptimalChunkSize
		infoLog.Printf("High latency network detected. Adjusting workers from %d to %d",
			originalWorkers, config.Workers)
	} else if profile.RTT < 10*time.Millisecond {
		// Low latency network, can use more workers
		config.Workers = min(runtime.NumCPU(), config.Workers*2)
		infoLog.Printf("Low latency network detected. Adjusting workers from %d to %d",
			originalWorkers, config.Workers)
	}

	if config.ChunkSize != originalChunkSize {
		infoLog.Printf("Adjusted chunk size from %.2f MB to %.2f MB based on network conditions",
			float64(originalChunkSize)/1024/1024, float64(config.ChunkSize)/1024/1024)
	}

	// Initialize transfer
	if err := writer.WriteByte(CMD_INIT); err != nil {
		errorLog.Fatalf("Failed to send init command: %v", err)
	}

	// Send filename
	if _, err := writer.WriteString(fileInfo.Name() + "\n"); err != nil {
		errorLog.Fatalf("Failed to send filename: %v", err)
	}

	// Send file size
	if _, err := writer.WriteString(fmt.Sprintf("%d\n", fileInfo.Size())); err != nil {
		errorLog.Fatalf("Failed to send file size: %v", err)
	}

	if err := writer.Flush(); err != nil {
		errorLog.Fatalf("Failed to flush init data: %v", err)
	}

	infoLog.Printf("Sending file: %s (%.2f MB)", fileInfo.Name(), float64(fileInfo.Size())/1024/1024)

	// Initialize stats
	stats := Stats{
		TotalBytes: fileInfo.Size(),
		StartTime:  time.Now(),
		FileSize:   fileInfo.Size(),
	}

	// Initialize network stats for adaptive chunk delays
	var netStats *NetworkStats
	if config.AdaptiveDelay {
		netStats = NewNetworkStats(config)
		infoLog.Printf("Adaptive network delay enabled (min: %v, max: %v)", netStats.minDelay, netStats.maxDelay)
	} else {
		// Create with default values but it won't be used in the same way
		netStats = NewNetworkStats(config)
		infoLog.Printf("Using fixed chunk delay: %v", config.ChunkDelay)
	}

	// Create buffer pool for chunks
	bufferPool := sync.Pool{
		New: func() interface{} {
			return make([]byte, config.ChunkSize)
		},
	}

	// Start progress reporting
	var progressDone chan struct{}
	if config.ShowProgress {
		progressDone = make(chan struct{})
		go reportProgress(&stats, progressDone)
	}

	// Handle chunk requests
	for {
		// We're using no timeout for the connection, so no need to reset deadlines
		// Individual operations might take time with large files, which is expected

		// Read command
		cmdByte, err := reader.ReadByte()
		if err != nil {
			errorLog.Fatalf("Failed to read command: %v", err)
		}

		switch cmdByte {
		case CMD_REQUEST:
			// Read chunk offset
			offsetStr, err := reader.ReadString('\n')
			if err != nil {
				errorLog.Fatalf("Failed to read chunk offset: %v", err)
			}
			offset, err := strconv.ParseInt(offsetStr[:len(offsetStr)-1], 10, 64)
			if err != nil {
				errorLog.Fatalf("Invalid chunk offset: %v", err)
			}

			// Apply adaptive delay based on network conditions
			chunkDelay := netStats.GetDelay(config.ChunkDelay)
			if chunkDelay > 0 {
				time.Sleep(chunkDelay)
			}

			// Handle chunk request
			buffer := bufferPool.Get().([]byte)

			actualChunkSize := config.ChunkSize
			if offset+config.ChunkSize > stats.FileSize {
				actualChunkSize = stats.FileSize - offset
			}

			if err := sendChunk(writer, file, offset, config.ChunkSize, buffer, &stats, config); err != nil {
				bufferPool.Put(buffer) // Return buffer to pool even on error
				errorLog.Fatalf("Failed to send chunk: %v", err)
			} else {
				// Update network stats after successful chunk transfer
				netStats.UpdateStats(actualChunkSize)
			}

			bufferPool.Put(buffer) // Return buffer to pool after successful use

		case CMD_HASH:
			// Calculate and send file hash
			if err := sendFileHash(writer, file); err != nil {
				errorLog.Fatalf("Failed to send file hash: %v", err)
			}

		case CMD_COMPLETE:
			// Transfer complete
			if config.ShowProgress {
				close(progressDone)
			}

			// Clean up transfer state file since the transfer is complete
			stateFilePath := filepath.Join(config.OutputDir, fileInfo.Name()+".justdatacopier.state")
			if err := os.Remove(stateFilePath); err != nil && !os.IsNotExist(err) {
				errorLog.Printf("Warning: Failed to remove transfer state file: %v", err)
				// Continue anyway, not critical
			}

			elapsed := time.Since(stats.StartTime)
			infoLog.Printf("File sent successfully: %s (%.2f MB in %v, %.2f MB/s)",
				fileInfo.Name(),
				float64(fileInfo.Size())/1024/1024,
				elapsed,
				float64(fileInfo.Size())/1024/1024/elapsed.Seconds())
			return

		case CMD_ERROR:
			// Read error message
			errorMsg, err := reader.ReadString('\n')
			if err != nil {
				errorLog.Fatalf("Failed to read error message: %v", err)
			}
			errorLog.Fatalf("Server error: %s", errorMsg[:len(errorMsg)-1])

		default:
			errorLog.Fatalf("Unknown command: %d", cmdByte)
		}
	}
}

func sendChunk(writer *bufio.Writer, file *os.File, offset int64, chunkSize int64, buffer []byte, stats *Stats, config Config) error {
	// Read chunk from file
	n, err := file.ReadAt(buffer, offset)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file chunk: %w", err)
	}

	// Create a context with a reasonable timeout for this chunk transfer
	// The timeout increases with chunk size to accommodate larger transfers
	timeoutPerMB := 10 * time.Second
	chunkSizeMB := float64(n) / (1024 * 1024)
	chunkTimeout := time.Duration(max(30, int(chunkSizeMB*timeoutPerMB.Seconds()))) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), chunkTimeout)
	defer cancel()

	// Send data command with retry
	maxRetries := 5 // Increased retries
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		// Check for context cancellation
		if ctx.Err() != nil {
			return fmt.Errorf("chunk send operation timed out: %w", ctx.Err())
		}

		// Add exponential backoff for retries
		if retry > 0 {
			backoffTime := time.Duration(retry*500) * time.Millisecond
			select {
			case <-time.After(backoffTime):
				// Continue after backoff
				infoLog.Printf("Retrying sending chunk at offset %d (attempt %d/%d)", offset, retry+1, maxRetries)
			case <-ctx.Done():
				return fmt.Errorf("chunk send operation timed out during backoff: %w", ctx.Err())
			}
		}

		// Start with a clean writer state for each attempt
		writer.Reset(writer)

		// Send data command
		if err := writer.WriteByte(CMD_DATA); err != nil {
			lastErr = fmt.Errorf("failed to send data command: %w", err)
			continue
		}

		// Send chunk size (ensure it ends with a newline)
		if _, err := fmt.Fprintf(writer, "%d\n", n); err != nil {
			lastErr = fmt.Errorf("failed to send chunk size: %w", err)
			continue
		}

		// Flush header information first and ensure it completes
		flushStartTime := time.Now()
		if err := writer.Flush(); err != nil {
			lastErr = fmt.Errorf("failed to flush chunk header: %w", err)
			continue
		}

		// If flush took a long time, log a warning as it might indicate network issues
		flushTime := time.Since(flushStartTime)
		if flushTime > time.Second {
			infoLog.Printf("Warning: Chunk header flush took %v - possible network congestion", flushTime)
		}

		// Handle compression if enabled
		if config.Compression {
			// Check if file should be compressed based on extension
			fileExt := strings.ToLower(filepath.Ext(file.Name()))
			alreadyCompressed := map[string]bool{
				".zip": true, ".gz": true, ".rar": true, ".7z": true,
				".mp3": true, ".mp4": true, ".jpg": true, ".jpeg": true,
				".png": true, ".gif": true, ".webp": true, ".pdf": true,
				".docx": true, ".xlsx": true, ".pptx": true}

			shouldCompress := !alreadyCompressed[fileExt]

			if shouldCompress {
				// Send compression flag (1 = compressed)
				if err := writer.WriteByte(1); err != nil {
					lastErr = fmt.Errorf("failed to send compression flag: %w", err)
					continue
				}

				// Create a buffer for compressed data
				var compBuf bytes.Buffer

				// Choose compression level based on file extension
				// Use BestSpeed for files that are already compressed or binary
				// Use DefaultCompression for text files that compress well
				compressionLevel := gzip.BestSpeed // Default to speed
				textExtensions := map[string]bool{".txt": true, ".log": true, ".csv": true, ".json": true, ".xml": true, ".html": true}

				if textExtensions[fileExt] {
					compressionLevel = gzip.DefaultCompression // Better compression for text
				}

				gzipWriter, err := gzip.NewWriterLevel(&compBuf, compressionLevel)
				if err != nil {
					lastErr = fmt.Errorf("failed to create gzip writer: %w", err)
					continue
				}

				// Compress the chunk
				if _, err := gzipWriter.Write(buffer[:n]); err != nil {
					gzipWriter.Close()
					lastErr = fmt.Errorf("failed to compress data: %w", err)
					continue
				}

				if err := gzipWriter.Close(); err != nil {
					lastErr = fmt.Errorf("failed to finalize compression: %w", err)
					continue
				}

				// Send compressed size
				compressedSize := compBuf.Len()
				if _, err := fmt.Fprintf(writer, "%d\n", compressedSize); err != nil {
					lastErr = fmt.Errorf("failed to send compressed size: %w", err)
					continue
				}

				// Flush compressed size header
				if err := writer.Flush(); err != nil {
					lastErr = fmt.Errorf("failed to flush compressed size: %w", err)
					continue
				}

				// Log compression ratio
				compressionRatio := float64(n) / float64(compressedSize)
				if retry == 0 { // Only log on first attempt to avoid spam
					infoLog.Printf("Chunk compressed: %d → %d bytes (%.2fx ratio)",
						n, compressedSize, compressionRatio)
				}

				// Send compressed data
				compressedData := compBuf.Bytes()

				// Send data in smaller pieces just like with uncompressed
				const maxWriteSize = 32 * 1024
				for i := 0; i < compressedSize && lastErr == nil; i += maxWriteSize {
					endPos := i + maxWriteSize
					if endPos > compressedSize {
						endPos = compressedSize
					}

					if _, err := writer.Write(compressedData[i:endPos]); err != nil {
						lastErr = fmt.Errorf("failed to send compressed data: %w", err)
						break
					}

					if err := writer.Flush(); err != nil {
						lastErr = fmt.Errorf("failed to flush compressed data: %w", err)
						break
					}
				}

				if lastErr != nil {
					continue
				}
			} else {
				// Skip compression for already compressed file types
				infoLog.Printf("Skipping compression for %s (already compressed format)", fileExt)

				// Send uncompressed flag (0 = uncompressed)
				if err := writer.WriteByte(0); err != nil {
					lastErr = fmt.Errorf("failed to send compression flag: %w", err)
					continue
				}

				// Flush compression flag
				if err := writer.Flush(); err != nil {
					lastErr = fmt.Errorf("failed to flush compression flag: %w", err)
					continue
				}

				// Send uncompressed data in smaller pieces
				const maxWriteSize = 32 * 1024 // Reduced to 32KB for better network performance
				var chunkSendFailed bool = false

				for i := 0; i < n && chunkSendFailed == false; i += maxWriteSize {
					endPos := i + maxWriteSize
					if endPos > n {
						endPos = n
					}

					// Set write deadline for each piece
					writeStart := time.Now()

					if _, err := writer.Write(buffer[i:endPos]); err != nil {
						lastErr = fmt.Errorf("failed to send chunk data at position %d: %w", i, err)
						chunkSendFailed = true
						break
					}

					// Flush after each piece with a timeout check
					if err := writer.Flush(); err != nil {
						lastErr = fmt.Errorf("failed to flush chunk data at position %d: %w", i, err)
						chunkSendFailed = true
						break
					}

					// Check if this piece took too long (possible slowdown detection)
					pieceTime := time.Since(writeStart)
					if pieceTime > 5*time.Second {
						infoLog.Printf("Warning: Slow network detected. Chunk piece took %v to send", pieceTime)
					}

					// Small sleep to allow OS to handle network buffers (can help on unstable networks)
					if i+maxWriteSize < n {
						time.Sleep(time.Millisecond * 5)
					}
				}

				// If this attempt failed, try again
				if chunkSendFailed {
					continue
				}
			}
		} else {
			// Send uncompressed flag (0 = uncompressed)
			if err := writer.WriteByte(0); err != nil {
				lastErr = fmt.Errorf("failed to send compression flag: %w", err)
				continue
			}

			// Flush compression flag
			if err := writer.Flush(); err != nil {
				lastErr = fmt.Errorf("failed to flush compression flag: %w", err)
				continue
			}

			// Send uncompressed data in smaller pieces with adaptive sizing
			// Start with a reasonable size and adjust based on performance
			maxWriteSize := 64 * 1024 // Start with 64KB chunks
			minWriteSize := 8 * 1024  // Don't go below 8KB for efficiency

			// Use context to control the overall operation time
			var chunkSendFailed bool = false
			consecutiveSlowWrites := 0

			for i := 0; i < n && !chunkSendFailed && ctx.Err() == nil; {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return fmt.Errorf("chunk send operation timed out: %w", ctx.Err())
				default:
					// Continue with send
				}

				// Calculate the end position for this piece
				endPos := i + maxWriteSize
				if endPos > n {
					endPos = n
				}

				pieceSize := endPos - i

				// Track write time for adaptive sizing
				writeStart := time.Now()

				// Write piece to buffer
				if _, err := writer.Write(buffer[i:endPos]); err != nil {
					lastErr = fmt.Errorf("failed to send chunk data at position %d: %w", i, err)
					chunkSendFailed = true
					break
				}

				// Flush after each piece with context check
				select {
				case <-ctx.Done():
					return fmt.Errorf("chunk send operation timed out during flush: %w", ctx.Err())
				default:
					if err := writer.Flush(); err != nil {
						lastErr = fmt.Errorf("failed to flush chunk data at position %d: %w", i, err)
						chunkSendFailed = true
						break
					}
				}

				// Check if this piece took too long and adapt the write size
				pieceTime := time.Since(writeStart)
				bytesPerSecond := float64(pieceSize) / pieceTime.Seconds()

				// If write was too slow, reduce size for next piece
				if pieceTime > 2*time.Second {
					infoLog.Printf("Warning: Slow network detected. Chunk piece took %v to send (%.2f KB/s)",
						pieceTime, bytesPerSecond/1024)

					// Reduce write size but don't go below minimum
					maxWriteSize = max(minWriteSize, maxWriteSize/2)
					consecutiveSlowWrites++

					// If we've had multiple slow writes, add a small pause to let network recover
					if consecutiveSlowWrites > 2 {
						pauseTime := 500 * time.Millisecond * time.Duration(consecutiveSlowWrites-2)
						if pauseTime > 5*time.Second {
							pauseTime = 5 * time.Second
						}
						infoLog.Printf("Multiple slow writes detected, pausing for %v to let network recover", pauseTime)
						time.Sleep(pauseTime)
					}
				} else if pieceTime < 200*time.Millisecond && maxWriteSize < 256*1024 {
					// Network is handling things well, try increasing write size up to a limit
					maxWriteSize = min(maxWriteSize*2, 256*1024)
					consecutiveSlowWrites = 0
				} else {
					// Reset consecutive slow count if we're in a good zone
					consecutiveSlowWrites = 0
				}

				// Move to next piece
				i = endPos
			}

			// Check for context cancellation
			if ctx.Err() != nil {
				return fmt.Errorf("chunk send operation timed out: %w", ctx.Err())
			}

			// If this attempt failed, try again
			if chunkSendFailed {
				continue
			}
		}

		// Final flush to ensure everything is sent
		if err := writer.Flush(); err != nil {
			lastErr = fmt.Errorf("failed to perform final flush: %w", err)
			continue
		}

		// If we get here, the chunk was sent successfully
		// Update stats
		stats.TransferredBytes.Add(int64(n))
		return nil
	}

	// If all retries failed, return the last error
	return fmt.Errorf("failed to send chunk after %d attempts: %w", maxRetries, lastErr)
}

func sendFileHash(writer *bufio.Writer, file *os.File) error {
	// Reset file position
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Calculate hash
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Send hash command
	if err := writer.WriteByte(CMD_HASH); err != nil {
		return fmt.Errorf("failed to send hash command: %w", err)
	}

	// Send hash
	if _, err := writer.WriteString(hex.EncodeToString(hash.Sum(nil)) + "\n"); err != nil {
		return fmt.Errorf("failed to send hash: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush hash: %w", err)
	}

	return nil
}

// UTILITY FUNCTIONS

func reportProgress(stats *Stats, done chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastTransferred int64
	var lastUpdateTime time.Time = time.Now()

	// For calculating moving average speed
	const speedWindowSize = 5
	speedHistory := make([]float64, 0, speedWindowSize)

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			transferred := stats.TransferredBytes.Load()
			percent := float64(transferred) / float64(stats.TotalBytes) * 100

			// Calculate current speed based on last update
			timeDiff := now.Sub(lastUpdateTime).Seconds()
			byteDiff := transferred - lastTransferred
			currentSpeed := float64(byteDiff) / 1024 / 1024 / timeDiff

			// Add to speed history for moving average
			speedHistory = append(speedHistory, currentSpeed)
			if len(speedHistory) > speedWindowSize {
				speedHistory = speedHistory[1:] // Remove oldest entry
			}

			// Calculate average speed
			var avgSpeed float64
			for _, s := range speedHistory {
				avgSpeed += s
			}
			avgSpeed /= float64(len(speedHistory))

			// Calculate ETA
			var eta string
			if avgSpeed > 0.1 { // Only show ETA if speed is reasonable
				remainingBytes := stats.TotalBytes - transferred
				remainingTime := float64(remainingBytes) / (avgSpeed * 1024 * 1024)

				if remainingTime < 60 {
					eta = fmt.Sprintf("%.0f sec", remainingTime)
				} else if remainingTime < 3600 {
					eta = fmt.Sprintf("%.1f min", remainingTime/60)
				} else {
					eta = fmt.Sprintf("%.1f hr", remainingTime/3600)
				}
			} else {
				eta = "calculating..."
			}

			// Create progress bar
			const barWidth = 30
			completedWidth := int(float64(barWidth) * percent / 100)
			progressBar := strings.Repeat("█", completedWidth) + strings.Repeat("░", barWidth-completedWidth)

			// Update display
			fmt.Printf("\r[%s] %.1f%% (%.2f/%.2f MB) at %.2f MB/s ETA: %s",
				progressBar,
				percent,
				float64(transferred)/1024/1024,
				float64(stats.TotalBytes)/1024/1024,
				avgSpeed,
				eta)

			// Update for next iteration
			lastTransferred = transferred
			lastUpdateTime = now

		case <-done:
			fmt.Println() // Print newline after progress
			return
		}
	}
}

func preallocateFile(file *os.File, size int64) error {
	// On Windows, Write() with zeros is more efficient than Truncate
	// This is a simplified version that works well enough for this use case
	const bufSize = 1024 * 1024 // 1MB buffer
	zeros := make([]byte, bufSize)

	remaining := size
	for remaining > 0 {
		writeSize := int64(bufSize)
		if remaining < writeSize {
			writeSize = remaining
		}

		n, err := file.Write(zeros[:writeSize])
		if err != nil {
			return err
		}
		remaining -= int64(n)
	}

	// Reset file position
	_, err := file.Seek(0, io.SeekStart)
	return err
}

// Helper functions for min/max operations that Go doesn't provide by default
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

// profileNetwork performs network profiling to determine optimal transfer parameters
func profileNetwork(conn net.Conn) NetworkProfile {
	profile := NetworkProfile{
		RTT:              100 * time.Millisecond, // Default values
		Bandwidth:        10 * 1024 * 1024,       // 10 MB/s default
		PacketLoss:       0.01,                   // Default
		OptimalChunkSize: 2 * 1024 * 1024,        // Default
	}

	// Create a context with timeout for profiling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a completely separate TCP connection just for profiling
	// This ensures we don't interfere with the main connection's state at all
	connAddr := conn.RemoteAddr().String()
	connType := conn.RemoteAddr().Network()

	infoLog.Printf("Creating separate profiling connection to %s", connAddr)

	profConn, err := net.DialTimeout(connType, connAddr, time.Second*5)
	if err != nil {
		infoLog.Printf("Failed to create profiling connection: %v, using default profile", err)
		return profile
	}
	defer profConn.Close()

	// Set proper timeouts
	profConn.SetDeadline(time.Now().Add(10 * time.Second))

	// Create buffers specifically for network profiling
	profReader := bufio.NewReader(profConn)
	profWriter := bufio.NewWriter(profConn)

	// Send ping packets to measure RTT
	var totalRTT time.Duration
	successfulPings := 0
	pingCount := 5 // Reduced from 10 to 5 to decrease profiling time

	for i := 0; i < pingCount; i++ {
		select {
		case <-ctx.Done():
			infoLog.Printf("Profiling timed out, using partial results")
			goto ProfileCompletion
		default:
			startTime := time.Now()

			// Send ping command
			if err := profWriter.WriteByte(CMD_PING); err != nil {
				infoLog.Printf("Ping write failed: %v", err)
				continue
			}

			if err := profWriter.Flush(); err != nil {
				infoLog.Printf("Ping flush failed: %v", err)
				continue
			}

			// Read response with deadline
			response, err := profReader.ReadByte()
			if err != nil {
				infoLog.Printf("Ping read response failed: %v", err)
				continue
			}

			if response != CMD_PONG {
				infoLog.Printf("Unexpected response to ping: %d", response)
				continue
			}

			// Calculate RTT
			rtt := time.Since(startTime)
			totalRTT += rtt
			successfulPings++

			// Small delay between pings
			time.Sleep(100 * time.Millisecond)
		}
	}

ProfileCompletion:
	// Calculate average RTT if we had any successful pings
	if successfulPings > 0 {
		profile.RTT = totalRTT / time.Duration(successfulPings)
	}

	// Log the RTT
	infoLog.Printf("Network profiling complete: RTT=%v (from %d successful pings)",
		profile.RTT, successfulPings)

	// Estimate bandwidth based on RTT (simple heuristic)
	// Lower RTT generally means higher bandwidth
	if profile.RTT < 10*time.Millisecond {
		profile.Bandwidth = 50 * 1024 * 1024 // 50 MB/s for very low latency
	} else if profile.RTT < 50*time.Millisecond {
		profile.Bandwidth = 20 * 1024 * 1024 // 20 MB/s for medium latency
	} else if profile.RTT < 100*time.Millisecond {
		profile.Bandwidth = 10 * 1024 * 1024 // 10 MB/s for high latency
	} else {
		profile.Bandwidth = 5 * 1024 * 1024 // 5 MB/s for very high latency
	}

	// Calculate optimal chunk size based on bandwidth-delay product (BDP)
	// BDP = bandwidth * RTT
	bdp := float64(profile.Bandwidth) * profile.RTT.Seconds()

	// Adjust chunk size to approximately BDP, but with limits
	optimalChunkSize := int64(bdp)
	if optimalChunkSize < 512*1024 {
		optimalChunkSize = 512 * 1024 // Minimum 512KB
	}
	if optimalChunkSize > 8*1024*1024 {
		optimalChunkSize = 8 * 1024 * 1024 // Maximum 8MB
	}

	profile.OptimalChunkSize = optimalChunkSize

	// Adjust for higher latency or packet loss by using larger chunks
	if profile.RTT > 50*time.Millisecond {
		// For higher latency, increase chunk size to reduce overhead
		increase := int64(float64(profile.OptimalChunkSize) * 1.5)
		if increase > 8*1024*1024 {
			increase = 8 * 1024 * 1024
		}
		profile.OptimalChunkSize = increase
	}

	return profile
}
