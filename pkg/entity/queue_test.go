package entity

import (
	"testing"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/stretchr/testify/require"
)

func TestQueueConfig_ToBytes_FromBytes_RoundTrip(t *testing.T) {
	orig := &QueueConfig{
		Name: "my-queue",
		Type: "fifo",
		Settings: QueueSettings{
			Strategy:   "ROUND_ROBIN",
			MaxUnacked: 7,
			AckTimeout: 45,
		},
	}

	data, err := orig.ToBytes()
	require.NoError(t, err, "ToBytes should not return error")

	decoded, err := QueueConfigFromBytes(data)
	require.NoError(t, err, "QueueConfigFromBytes should not return error for valid proto bytes")

	require.Equal(t, orig.Name, decoded.Name)
	require.Equal(t, orig.Type, decoded.Type)
	require.Equal(t, orig.Settings.Strategy, decoded.Settings.Strategy)
	require.Equal(t, orig.Settings.MaxUnacked, decoded.Settings.MaxUnacked)
	require.Equal(t, orig.Settings.AckTimeout, decoded.Settings.AckTimeout)
}

func TestQueueConfig_ToProto_StrategyMapping(t *testing.T) {
	qc := &QueueConfig{
		Name: "s",
		Type: "t",
		Settings: QueueSettings{
			Strategy:   "WEIGHTED",
			MaxUnacked: 3,
			AckTimeout: 10,
		},
	}

	protoQ := qc.ToProto()
	require.NotNil(t, protoQ)
	require.NotNil(t, protoQ.Settings)

	require.Equal(t, pb.QueueSettings_WEIGHTED, protoQ.Settings.Strategy)
	require.Equal(t, uint32(qc.Settings.MaxUnacked), protoQ.Settings.MaxUnacked)
	require.Equal(t, qc.Settings.AckTimeout, protoQ.Settings.AckTimeout)
}

// TestQueueConfigDeliveryLimitRoundTrip checks DeliveryLimit survives the
// stored-bytes round trip, and that 0 (unlimited) is the default for configs
// written before the setting existed.
func TestQueueConfigDeliveryLimitRoundTrip(t *testing.T) {
	qc := &QueueConfig{
		Name: "test_queue",
		Type: "fair",
		Settings: QueueSettings{
			Strategy:      "WEIGHTED",
			MaxUnacked:    5,
			AckTimeout:    60,
			DeliveryLimit: 3,
		},
	}

	data, err := qc.ToBytes()
	require.NoError(t, err)

	decoded, err := QueueConfigFromBytes(data)
	require.NoError(t, err)
	require.Equal(t, uint32(3), decoded.Settings.DeliveryLimit)
	require.Equal(t, qc, decoded)

	bare := &QueueConfig{Name: "q", Type: "delayed", Settings: QueueSettings{Strategy: "WEIGHTED"}}
	data, err = bare.ToBytes()
	require.NoError(t, err)
	decoded, err = QueueConfigFromBytes(data)
	require.NoError(t, err)
	require.Equal(t, uint32(0), decoded.Settings.DeliveryLimit)
}
