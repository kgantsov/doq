package entity

import (
	"encoding/json"
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
	return json.Marshal(m)
}

func (m *Message) UpdatePriority(newPriority int64) {
	m.Priority = newPriority
}

func MessageFromBytes(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
