package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	enqueueTotal    *prometheus.CounterVec
	dequeueTotal    *prometheus.CounterVec
	ackTotal        *prometheus.CounterVec
	nackTotal       *prometheus.CounterVec
	messages        *prometheus.GaugeVec
	unackedMessages *prometheus.GaugeVec
	readyMessages   *prometheus.GaugeVec
	registry        *prometheus.Registry
}

func NewPrometheusMetrics(registry prometheus.Registerer, namespace, subsystem string) *PrometheusMetrics {
	m := &PrometheusMetrics{}

	m.enqueueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "enqueue_total",
			Help:      "Total number of enqueued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.enqueueTotal)

	m.dequeueTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "dequeue_total",
			Help:      "Total number of dequeued messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.dequeueTotal)

	m.ackTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ack_total",
			Help:      "Total number of acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.ackTotal)

	m.nackTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "nack_total",
			Help:      "Total number of not acknowledged messages.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.nackTotal)

	m.messages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages",
			Help:      "Number of messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.messages)

	m.unackedMessages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "unacked_messages",
			Help:      "Number of unacknowledged messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.unackedMessages)

	m.readyMessages = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "ready_messages",
			Help:      "Number of ready messages in the queue.",
		},
		[]string{"queue_name"},
	)
	prometheus.MustRegister(m.readyMessages)

	return m
}
