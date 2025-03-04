package metrics

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
	assert.NotNil(t, metrics.EnqueueTotal)
	assert.NotNil(t, metrics.EnqueueTotal)
	assert.NotNil(t, metrics.DequeueTotal)
	assert.NotNil(t, metrics.AckTotal)
	assert.NotNil(t, metrics.NackTotal)
	assert.NotNil(t, metrics.Messages)
	assert.NotNil(t, metrics.UnackedMessages)
	assert.NotNil(t, metrics.ReadyMessages)
}
