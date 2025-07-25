package grpc

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type PrometheusMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec

	Registry *prometheus.Registry
}

func NewPrometheusMetrics(registry prometheus.Registerer, namespace, subsystem string) *PrometheusMetrics {
	m := &PrometheusMetrics{}

	m.RequestsTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total number of gRPC requests",
		},
		[]string{"method", "code"},
	)
	prometheus.MustRegister(m.RequestsTotal)

	m.RequestDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of gRPC requests in seconds",
			Buckets: []float64{
				0.000000001, // 1ns
				0.000000002,
				0.000000005,
				0.00000001, // 10ns
				0.00000002,
				0.00000005,
				0.0000001, // 100ns
				0.0000002,
				0.0000005,
				0.000001, // 1µs
				0.000002,
				0.000005,
				0.00001, // 10µs
				0.00002,
				0.00005,
				0.0001, // 100µs
				0.0002,
				0.0005,
				0.001, // 1ms
				0.002,
				0.005,
				0.01, // 10ms
				0.02,
				0.05,
				0.1, // 100 ms
				0.2,
				0.5,
				1.0, // 1s
				2.0,
				5.0,
				10.0, // 10s
				15.0,
				20.0,
				30.0,
				60.0, // 1m
			},
		},
		[]string{"method"},
	)
	prometheus.MustRegister(m.RequestDuration)

	return m
}

func UnaryInterceptor(prometheusEnabled bool, m *PrometheusMetrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		duration := time.Since(start)

		code := status.Code(err)

		if prometheusEnabled {
			m.RequestsTotal.WithLabelValues(info.FullMethod, code.String()).Inc()
			m.RequestDuration.WithLabelValues(info.FullMethod).Observe(duration.Seconds())
		}

		log.Info().
			Dur("duration", duration).
			Str("code", code.String()).
			Msgf("[grpc] %s %s %s", info.FullMethod, code.String(), duration)

		return resp, err
	}
}

func StreamInterceptor(prometheusEnabled bool, m *PrometheusMetrics) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start)

		code := status.Code(err)

		if prometheusEnabled {
			m.RequestsTotal.WithLabelValues(info.FullMethod, code.String()).Inc()
			m.RequestDuration.WithLabelValues(info.FullMethod).Observe(duration.Seconds())
		}

		log.Info().
			Dur("duration", duration).
			Str("code", code.String()).
			Msgf("[grpc] %s %s %s", info.FullMethod, code.String(), duration)

		return err
	}
}
