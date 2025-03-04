package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	EnqueueTotal    *prometheus.CounterVec
	DequeueTotal    *prometheus.CounterVec
	AckTotal        *prometheus.CounterVec
	NackTotal       *prometheus.CounterVec
	Messages        *prometheus.GaugeVec
	UnackedMessages *prometheus.GaugeVec
	ReadyMessages   *prometheus.GaugeVec
	Registry        *prometheus.Registry
}

func NewPrometheusMetrics(registry prometheus.Registerer, namespace, subsystem string) *PrometheusMetrics {
	m := &PrometheusMetrics{}

	m.EnqueueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "enqueue_total",
			Help:      "Total number of enqueued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.EnqueueTotal)

	m.DequeueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "dequeue_total",
			Help:      "Total number of dequeued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.DequeueTotal)

	m.AckTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ack_total",
			Help:      "Total number of acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.AckTotal)

	m.NackTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "nack_total",
			Help:      "Total number of not acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.NackTotal)

	m.Messages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages",
			Help:      "Number of messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.Messages)

	m.UnackedMessages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "unacked_messages",
			Help:      "Number of unacknowledged messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.UnackedMessages)

	m.ReadyMessages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ready_messages",
			Help:      "Number of ready messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.ReadyMessages)

	return m
}
