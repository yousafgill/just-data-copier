package network

import (
	"testing"
	"time"

	"justdatacopier/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkStats(t *testing.T) {
	cfg := &config.Config{
		AdaptiveDelay: true,
		MinDelay:      time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
	}

	stats := NewNetworkStats(cfg)

	assert.NotNil(t, stats)
	assert.Equal(t, 1.0, stats.DelayMultiplier)
	assert.NotZero(t, stats.LastChunkTime)
}

func TestUpdateStats(t *testing.T) {
	cfg := &config.Config{
		AdaptiveDelay: true,
		MinDelay:      time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
	}

	stats := NewNetworkStats(cfg)

	// Wait a small amount to ensure time difference
	time.Sleep(time.Millisecond)

	// Test updating stats with chunk data
	chunkSize := int64(1024)
	stats.UpdateStats(chunkSize)

	assert.Equal(t, chunkSize, stats.LastChunkSize)
	assert.True(t, stats.AvgTransferRate > 0)
}

func TestGetDelay(t *testing.T) {
	cfg := &config.Config{
		AdaptiveDelay: true,
		MinDelay:      time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		ChunkDelay:    10 * time.Millisecond,
	}

	stats := NewNetworkStats(cfg)

	baseDelay := 10 * time.Millisecond
	delay := stats.GetDelay(baseDelay)

	// Should return a delay within the configured range
	assert.True(t, delay >= cfg.MinDelay)
	assert.True(t, delay <= cfg.MaxDelay)
}

func TestGetDelayWithoutAdaptive(t *testing.T) {
	cfg := &config.Config{
		AdaptiveDelay: false,
		ChunkDelay:    10 * time.Millisecond,
	}

	stats := NewNetworkStats(cfg)

	baseDelay := 10 * time.Millisecond
	delay := stats.GetDelay(baseDelay)

	// Should return the base delay when adaptive is disabled
	assert.Equal(t, baseDelay, delay)
}
