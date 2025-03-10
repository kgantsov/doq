package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQueueStats(t *testing.T) {
	stats := NewQueueStats(10)
	go stats.Start()
	assert.Equal(t, 10, stats.windowSize)
	assert.NotNil(t, stats.quit)
	assert.NotNil(t, stats.enqueueHistory)
	assert.NotNil(t, stats.dequeueHistory)
	assert.NotNil(t, stats.ackHistory)
	assert.NotNil(t, stats.nackHistory)

	rps := stats.GetRPS()

	assert.Equal(t, 0.0, rps.AckRPS)
	assert.Equal(t, 0.0, rps.DequeueRPS)
	assert.Equal(t, 0.0, rps.EnqueueRPS)
	assert.Equal(t, 0.0, rps.NackRPS)

	for i := 0; i < 10; i++ {
		stats.IncrementEnqueue()
	}

	for i := 0; i < 5; i++ {
		stats.IncrementDequeue()
	}

	for i := 0; i < 3; i++ {
		stats.IncrementAck()
	}
	for i := 0; i < 2; i++ {
		stats.IncrementNack()
	}

	stats.UpdateWindow()

	rps = stats.GetRPS()

	assert.Equal(t, 0.3, rps.AckRPS)
	assert.Equal(t, 0.5, rps.DequeueRPS)
	assert.Equal(t, 1.0, rps.EnqueueRPS)
	assert.Equal(t, 0.2, rps.NackRPS)

	stats.Stop()
}

func TestQueueStats_GetRPS(t *testing.T) {
	tests := []struct {
		name     string
		enqueue  uint64
		dequeue  uint64
		ack      uint64
		nack     uint64
		expected Stats
	}{
		{
			name:    "All zero",
			enqueue: 0,
			dequeue: 0,
			ack:     0,
			nack:    0,
			expected: Stats{
				EnqueueRPS: 0.0,
				DequeueRPS: 0.0,
				AckRPS:     0.0,
				NackRPS:    0.0,
			},
		},
		{
			name:    "All non-zero",
			enqueue: 10,
			dequeue: 5,
			ack:     3,
			nack:    2,
			expected: Stats{
				EnqueueRPS: 1.0,
				DequeueRPS: 0.5,
				AckRPS:     0.3,
				NackRPS:    0.2,
			},
		},
		{
			name:    "Large numbers",
			enqueue: 5000,
			dequeue: 2500,
			ack:     3561,
			nack:    2140,
			expected: Stats{
				EnqueueRPS: 500.0,
				DequeueRPS: 250.0,
				AckRPS:     356.1,
				NackRPS:    214.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewQueueStats(10)
			stats.enqueueCount = tt.enqueue
			stats.dequeueCount = tt.dequeue
			stats.ackCount = tt.ack
			stats.nackCount = tt.nack

			stats.UpdateWindow()

			rps := stats.GetRPS()
			assert.Equal(t, tt.expected.AckRPS, rps.AckRPS)
			assert.Equal(t, tt.expected.DequeueRPS, rps.DequeueRPS)
			assert.Equal(t, tt.expected.EnqueueRPS, rps.EnqueueRPS)
			assert.Equal(t, tt.expected.NackRPS, rps.NackRPS)
		})
	}
}
