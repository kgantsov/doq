package queue

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	namespace := "doq"
	subsystem := "queues"

	metrics := NewPrometheusMetrics(registry, namespace, subsystem)

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.enqueueTotal)
	assert.NotNil(t, metrics.enqueueTotal)
	assert.NotNil(t, metrics.dequeueTotal)
	assert.NotNil(t, metrics.ackTotal)
	assert.NotNil(t, metrics.nackTotal)
	assert.NotNil(t, metrics.messages)
	assert.NotNil(t, metrics.unackedMessages)
	assert.NotNil(t, metrics.readyMessages)
}
