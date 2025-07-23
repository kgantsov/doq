package entity

import (
	pb "github.com/kgantsov/doq/pkg/proto"
	"google.golang.org/protobuf/proto"
)

type Message struct {
	Group    string
	ID       uint64
	Priority int64
	Content  string
	Metadata map[string]string

	QueueName     string        `json:"QueueName,omitempty"`
	QueueType     string        `json:"QueueType,omitempty"`
	QueueSettings QueueSettings `json:"QueueSettings,omitempty"`
}

func (m *Message) ToBytes() ([]byte, error) {
	msg := m.ToProto()
	return proto.Marshal(msg)
}

func (m *Message) ToProto() *pb.Message {
	return &pb.Message{
		Group:    m.Group,
		Id:       m.ID,
		Priority: m.Priority,
		Content:  m.Content,
		Metadata: m.Metadata,
	}
}

func (m *Message) UpdatePriority(newPriority int64) {
	m.Priority = newPriority
}

func MessageFromBytes(data []byte) (*Message, error) {
	var msg pb.Message
	err := proto.Unmarshal(data, &msg)
	return &Message{
		Group:    msg.Group,
		ID:       msg.Id,
		Priority: msg.Priority,
		Content:  msg.Content,
		Metadata: msg.Metadata,
	}, err
}

func MessageProtoFromBytes(data []byte) (*pb.Message, error) {
	var msg pb.Message
	err := proto.Unmarshal(data, &msg)
	return &msg, err
}
