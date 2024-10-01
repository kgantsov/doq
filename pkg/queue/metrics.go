package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type QueueMetrics struct {
	EnqueueTotal float64
	DequeueTotal float64
	AckTotal     float64
	NackTotal    float64
	QueueSize    float64
}

type Metrics struct {
	enqueueTotal *prometheus.CounterVec
	dequeueTotal *prometheus.CounterVec
	ackTotal     *prometheus.CounterVec
	nackTotal    *prometheus.CounterVec
	queueSize    *prometheus.GaugeVec
	registry     *prometheus.Registry
}

func NewMetrics(registry prometheus.Registerer, subsystem string) *Metrics {

	m := &Metrics{}

	m.enqueueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "enqueue_total",
			Help:      "Total number of enqueued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.enqueueTotal)

	m.dequeueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "dequeue_total",
			Help:      "Total number of dequeued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.dequeueTotal)

	m.ackTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "ack_total",
			Help:      "Total number of acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.ackTotal)

	m.nackTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "nack_total",
			Help:      "Total number of not acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.nackTotal)

	m.queueSize = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "queue_size",
			Help:      "Current size of the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.queueSize)

	return m
}
