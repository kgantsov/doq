package entity

import (
	pb "github.com/kgantsov/doq/pkg/proto"
	"google.golang.org/protobuf/proto"
)

type QueueSettings struct {
	Strategy   string `json:"strategy,omitempty"`
	MaxUnacked int    `json:"max_unacked,omitempty"`
}

type QueueConfig struct {
	Name     string
	Type     string
	Settings QueueSettings
}

func (qc *QueueConfig) ToBytes() ([]byte, error) {
	queue := qc.ToProto()
	return proto.Marshal(queue)
}

func (qc *QueueConfig) ToProto() *pb.Queue {
	return &pb.Queue{
		Name: qc.Name,
		Type: qc.Type,
		Settings: &pb.QueueSettings{
			Strategy: pb.QueueSettings_Strategy(
				pb.QueueSettings_Strategy_value[qc.Settings.Strategy],
			),
			MaxUnacked: uint32(qc.Settings.MaxUnacked),
		},
	}
}

func QueueConfigFromBytes(data []byte) (*QueueConfig, error) {
	var qc pb.Queue
	err := proto.Unmarshal(data, &qc)
	return &QueueConfig{
		Name: qc.Name,
		Type: qc.Type,
		Settings: QueueSettings{
			Strategy:   qc.Settings.Strategy.String(),
			MaxUnacked: int(qc.Settings.MaxUnacked),
		},
	}, err
}
