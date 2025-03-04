package metrics

import (
	"sync/atomic"
	"time"
)

type Stats struct {
	EnqueueRPS float64
	DequeueRPS float64
	AckRPS     float64
	NackRPS    float64
}

type QueueStats struct {
	enqueueCount uint64
	dequeueCount uint64
	ackCount     uint64
	nackCount    uint64

	enqueueHistory []uint64
	dequeueHistory []uint64
	ackHistory     []uint64
	nackHistory    []uint64
	windowSize     int

	quit chan struct{}
}

func NewQueueStats(windowSize int) *QueueStats {
	stats := &QueueStats{
		enqueueHistory: make([]uint64, windowSize),
		dequeueHistory: make([]uint64, windowSize),
		ackHistory:     make([]uint64, windowSize),
		nackHistory:    make([]uint64, windowSize),
		windowSize:     windowSize,
		quit:           make(chan struct{}),
	}

	return stats
}

func (rc *QueueStats) Start() {
	// Ticker to update window every second
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.UpdateWindow()
		case <-rc.quit:
			return
		}
	}
}

func (rc *QueueStats) Stop() {
	close(rc.quit)
}

func (rc *QueueStats) IncrementEnqueue() {
	atomic.AddUint64(&rc.enqueueCount, 1)
}

func (rc *QueueStats) IncrementDequeue() {
	atomic.AddUint64(&rc.dequeueCount, 1)
}

func (rc *QueueStats) IncrementAck() {
	atomic.AddUint64(&rc.ackCount, 1)
}
func (rc *QueueStats) IncrementNack() {
	atomic.AddUint64(&rc.nackCount, 1)
}

func (rc *QueueStats) UpdateWindow() {
	// Shift history to the left and store current counts
	for i := 1; i < rc.windowSize; i++ {
		rc.enqueueHistory[i-1] = rc.enqueueHistory[i]
		rc.dequeueHistory[i-1] = rc.dequeueHistory[i]
		rc.ackHistory[i-1] = rc.ackHistory[i]
		rc.nackHistory[i-1] = rc.nackHistory[i]
	}
	rc.enqueueHistory[rc.windowSize-1] = atomic.SwapUint64(&rc.enqueueCount, 0)
	rc.dequeueHistory[rc.windowSize-1] = atomic.SwapUint64(&rc.dequeueCount, 0)
	rc.ackHistory[rc.windowSize-1] = atomic.SwapUint64(&rc.ackCount, 0)
	rc.nackHistory[rc.windowSize-1] = atomic.SwapUint64(&rc.nackCount, 0)
}

func (rc *QueueStats) GetRPS() *Stats {

	var totalEnqueue, totalDequeue, totalAck, totalNack uint64

	// Sum over the window
	for i := 0; i < rc.windowSize; i++ {
		totalEnqueue += rc.enqueueHistory[i]
		totalDequeue += rc.dequeueHistory[i]
		totalAck += rc.ackHistory[i]
		totalNack += rc.nackHistory[i]
	}

	// Calculate average rate (requests per second)
	seconds := float64(rc.windowSize)
	return &Stats{
		EnqueueRPS: float64(totalEnqueue) / seconds,
		DequeueRPS: float64(totalDequeue) / seconds,
		AckRPS:     float64(totalAck) / seconds,
		NackRPS:    float64(totalNack) / seconds,
	}
}
