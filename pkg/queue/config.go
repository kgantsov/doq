package queue

import "encoding/json"

type QueueConfig struct {
	Name string
	Type string
}

func (qc *QueueConfig) ToBytes() ([]byte, error) {
	return json.Marshal(qc)
}

func QueueConfigFromBytes(data []byte) (*QueueConfig, error) {
	var qc QueueConfig
	err := json.Unmarshal(data, &qc)
	return &qc, err
}
