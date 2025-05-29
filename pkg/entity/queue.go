package entity

import "encoding/json"

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
	return json.Marshal(qc)
}

func QueueConfigFromBytes(data []byte) (*QueueConfig, error) {
	var qc QueueConfig
	err := json.Unmarshal(data, &qc)
	return &qc, err
}
