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
	"log/slog"
	"os"
	"os/signal"
	"runtime"
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
