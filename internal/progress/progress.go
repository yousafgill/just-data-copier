package progress

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"justdatacopier/internal/logging"
)

// Stats holds transfer statistics
type Stats struct {
	TotalBytes       int64
	TransferredBytes atomic.Int64
	StartTime        time.Time
	FileSize         int64
	Filename         string
}

// Reporter handles progress reporting
type Reporter struct {
	stats       *Stats
	ticker      *time.Ticker
	done        chan struct{}
	showConsole bool
}

// NewReporter creates a new progress reporter
func NewReporter(stats *Stats, showConsole bool) *Reporter {
	return &Reporter{
		stats:       stats,
		ticker:      time.NewTicker(1 * time.Second),
		done:        make(chan struct{}),
		showConsole: showConsole,
	}
}

// Start begins progress reporting
func (r *Reporter) Start() {
	go r.reportLoop()
}

// Stop stops progress reporting
func (r *Reporter) Stop() {
	r.ticker.Stop()
	close(r.done)
	if r.showConsole {
		fmt.Println() // Print newline after progress bar
	}
}

// reportLoop runs the progress reporting loop
func (r *Reporter) reportLoop() {
	var lastTransferred int64
	var lastUpdateTime = time.Now()

	// For calculating moving average speed
	const speedWindowSize = 5
	speedHistory := make([]float64, 0, speedWindowSize)

	for {
		select {
		case <-r.ticker.C:
			r.updateProgress(&lastTransferred, &lastUpdateTime, &speedHistory)
		case <-r.done:
			return
		}
	}
}

// updateProgress updates and displays current progress
func (r *Reporter) updateProgress(lastTransferred *int64, lastUpdateTime *time.Time, speedHistory *[]float64) {
	now := time.Now()
	transferred := r.stats.TransferredBytes.Load()
	percent := float64(transferred) / float64(r.stats.TotalBytes) * 100

	// Calculate current speed based on last update
	timeDiff := now.Sub(*lastUpdateTime).Seconds()
	byteDiff := transferred - *lastTransferred
	currentSpeed := float64(byteDiff) / 1024 / 1024 / timeDiff

	// Add to speed history for moving average
	*speedHistory = append(*speedHistory, currentSpeed)
	if len(*speedHistory) > 5 { // speedWindowSize
		*speedHistory = (*speedHistory)[1:] // Remove oldest entry
	}

	// Calculate average speed
	var avgSpeed float64
	for _, s := range *speedHistory {
		avgSpeed += s
	}
	if len(*speedHistory) > 0 {
		avgSpeed /= float64(len(*speedHistory))
	}

	// Calculate ETA
	var eta string
	if avgSpeed > 0.1 { // Only show ETA if speed is reasonable
		remainingBytes := r.stats.TotalBytes - transferred
		remainingTime := float64(remainingBytes) / (avgSpeed * 1024 * 1024)

		switch {
		case remainingTime < 60:
			eta = fmt.Sprintf("%.0f sec", remainingTime)
		case remainingTime < 3600:
			eta = fmt.Sprintf("%.1f min", remainingTime/60)
		default:
			eta = fmt.Sprintf("%.1f hr", remainingTime/3600)
		}
	} else {
		eta = "calculating..."
	}

	// Log progress periodically (every 10 seconds)
	if int(now.Sub(r.stats.StartTime).Seconds())%10 == 0 {
		logging.LogTransferProgress(r.stats.Filename, transferred, r.stats.TotalBytes, avgSpeed)
	}

	// Show console progress if enabled
	if r.showConsole {
		r.showConsoleProgress(percent, transferred, avgSpeed, eta)
	}

	// Update for next iteration
	*lastTransferred = transferred
	*lastUpdateTime = now
}

// showConsoleProgress displays progress bar in console
func (r *Reporter) showConsoleProgress(percent float64, transferred int64, avgSpeed float64, eta string) {
	// Create progress bar
	const barWidth = 30
	completedWidth := int(float64(barWidth) * percent / 100)
	progressBar := strings.Repeat("█", completedWidth) + strings.Repeat("░", barWidth-completedWidth)

	// Update display
	fmt.Printf("\r[%s] %.1f%% (%.2f/%.2f MB) at %.2f MB/s ETA: %s",
		progressBar,
		percent,
		float64(transferred)/1024/1024,
		float64(r.stats.TotalBytes)/1024/1024,
		avgSpeed,
		eta)
}

// GetCurrentStats returns current transfer statistics
func (r *Reporter) GetCurrentStats() (transferred int64, percent float64, elapsed time.Duration) {
	transferred = r.stats.TransferredBytes.Load()
	percent = float64(transferred) / float64(r.stats.TotalBytes) * 100
	elapsed = time.Since(r.stats.StartTime)
	return
}

// UpdateTransferred atomically updates the transferred bytes count
func (s *Stats) UpdateTransferred(bytes int64) {
	s.TransferredBytes.Add(bytes)
}

// GetTransferred atomically gets the current transferred bytes count
func (s *Stats) GetTransferred() int64 {
	return s.TransferredBytes.Load()
}

// SetTransferred atomically sets the transferred bytes count
func (s *Stats) SetTransferred(bytes int64) {
	s.TransferredBytes.Store(bytes)
}
