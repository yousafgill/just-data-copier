package network

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"justdatacopier/internal/config"
	"justdatacopier/internal/errors"
	"justdatacopier/internal/protocol"
)

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
func NewNetworkStats(cfg *config.Config) *NetworkStats {
	minDelay := config.DefaultMinDelay
	maxDelay := config.DefaultMaxDelay

	// Use config values if adaptive delay is enabled
	if cfg.AdaptiveDelay {
		minDelay = cfg.MinDelay
		maxDelay = cfg.MaxDelay
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
		prevMultiplier := ns.DelayMultiplier

		// Smooth the rate with exponential moving average
		if ns.AvgTransferRate == 0 {
			ns.AvgTransferRate = currentRate
		} else {
			ns.AvgTransferRate = 0.7*ns.AvgTransferRate + 0.3*currentRate
		}

		// Adjust delay multiplier based on transfer rate
		if currentRate < 0.7*ns.AvgTransferRate {
			ns.DelayMultiplier *= 1.2
		} else if currentRate > 1.2*ns.AvgTransferRate {
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
			currentRateMB := currentRate / (1024 * 1024)
			avgRateMB := ns.AvgTransferRate / (1024 * 1024)

			if ns.DelayMultiplier > prevMultiplier {
				slog.Info("Network congestion detected",
					"current_rate_mbps", fmt.Sprintf("%.2f", currentRateMB),
					"avg_rate_mbps", fmt.Sprintf("%.2f", avgRateMB),
					"delay_factor", fmt.Sprintf("%.1f", ns.DelayMultiplier))
			} else {
				slog.Info("Network improving",
					"current_rate_mbps", fmt.Sprintf("%.2f", currentRateMB),
					"avg_rate_mbps", fmt.Sprintf("%.2f", avgRateMB),
					"delay_factor", fmt.Sprintf("%.1f", ns.DelayMultiplier))
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

	return delay
}

// ProfileNetwork performs network profiling to determine optimal transfer parameters
func ProfileNetwork(conn net.Conn) NetworkProfile {
	profile := NetworkProfile{
		RTT:              100 * time.Millisecond,  // Default values
		Bandwidth:        10 * 1024 * 1024,        // 10 MB/s default
		PacketLoss:       0.01,                    // Default
		OptimalChunkSize: config.DefaultChunkSize, // Default
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ProfileTimeout)
	defer cancel()

	connAddr := conn.RemoteAddr().String()
	connType := conn.RemoteAddr().Network()

	slog.Info("Creating profiling connection", "address", connAddr)

	profConn, err := net.DialTimeout(connType, connAddr, 5*time.Second)
	if err != nil {
		slog.Warn("Failed to create profiling connection, using defaults", "error", err)
		return profile
	}
	defer profConn.Close()

	// Set proper timeouts
	profConn.SetDeadline(time.Now().Add(10 * time.Second))

	profReader := bufio.NewReader(profConn)
	profWriter := bufio.NewWriter(profConn)

	// Send ping packets to measure RTT
	var totalRTT time.Duration
	successfulPings := 0

	for i := 0; i < config.PingCount; i++ {
		select {
		case <-ctx.Done():
			slog.Info("Profiling timed out, using partial results")
			goto ProfileCompletion
		default:
			startTime := time.Now()

			// Send ping command
			if err := protocol.SendCommand(profWriter, protocol.CmdPing); err != nil {
				slog.Debug("Ping write failed", "error", err)
				continue
			}

			if err := protocol.FlushWriter(profWriter); err != nil {
				slog.Debug("Ping flush failed", "error", err)
				continue
			}

			// Read response with deadline
			response, err := protocol.ReadCommand(ctx, profReader)
			if err != nil {
				slog.Debug("Ping read response failed", "error", err)
				continue
			}

			if response != protocol.CmdPong {
				slog.Debug("Unexpected response to ping", "response", response)
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

	slog.Info("Network profiling complete",
		"rtt", profile.RTT,
		"successful_pings", successfulPings)

	// Estimate bandwidth based on RTT (simple heuristic)
	switch {
	case profile.RTT < 10*time.Millisecond:
		profile.Bandwidth = 50 * 1024 * 1024 // 50 MB/s for very low latency
	case profile.RTT < 50*time.Millisecond:
		profile.Bandwidth = 20 * 1024 * 1024 // 20 MB/s for medium latency
	case profile.RTT < 100*time.Millisecond:
		profile.Bandwidth = 10 * 1024 * 1024 // 10 MB/s for high latency
	default:
		profile.Bandwidth = 5 * 1024 * 1024 // 5 MB/s for very high latency
	}

	// Calculate optimal chunk size based on bandwidth-delay product (BDP)
	bdp := float64(profile.Bandwidth) * profile.RTT.Seconds()
	optimalChunkSize := int64(bdp)

	// Apply limits
	if optimalChunkSize < 512*1024 {
		optimalChunkSize = 512 * 1024 // Minimum 512KB
	}
	if optimalChunkSize > 8*1024*1024 {
		optimalChunkSize = 8 * 1024 * 1024 // Maximum 8MB
	}

	profile.OptimalChunkSize = optimalChunkSize

	// Adjust for higher latency
	if profile.RTT > 50*time.Millisecond {
		increase := int64(float64(profile.OptimalChunkSize) * 1.5)
		if increase <= 8*1024*1024 {
			profile.OptimalChunkSize = increase
		}
	}

	return profile
}

// OptimizeTCPConnection applies TCP optimizations to a connection
func OptimizeTCPConnection(conn net.Conn) error {
	tcpConn, isTCP := conn.(*net.TCPConn)
	if !isTCP {
		return nil // Not a TCP connection, skip optimizations
	}

	// Enable keep-alive to detect dead connections
	if err := tcpConn.SetKeepAlive(true); err != nil {
		return errors.NewNetworkError("set_keepalive", conn.RemoteAddr().String(), err)
	}

	// Set keep-alive interval
	if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
		slog.Warn("Failed to set TCP keepalive period", "error", err)
	}

	// Disable Nagle's algorithm for better performance with chunking
	if err := tcpConn.SetNoDelay(true); err != nil {
		slog.Warn("Failed to disable Nagle's algorithm", "error", err)
	}

	// Set larger buffer sizes for high throughput
	if err := tcpConn.SetReadBuffer(config.TCPBufferSize); err != nil {
		slog.Warn("Failed to set TCP read buffer", "error", err)
	}

	if err := tcpConn.SetWriteBuffer(config.TCPBufferSize); err != nil {
		slog.Warn("Failed to set TCP write buffer", "error", err)
	}

	return nil
}
