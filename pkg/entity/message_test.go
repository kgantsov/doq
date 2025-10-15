package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_ToProto(t *testing.T) {
	m := &Message{
		Group:    "group-1",
		ID:       42,
		Priority: 7,
		Content:  "hello",
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	p := m.ToProto()
	assert.NotNil(t, p)
	assert.Equal(t, m.Group, p.Group)
	assert.Equal(t, m.ID, p.Id)
	assert.Equal(t, m.Priority, p.Priority)
	assert.Equal(t, m.Content, p.Content)
	assert.Equal(t, m.Metadata, p.Metadata)

	pb, err := m.ToBytes()
	assert.NoError(t, err)
	assert.NotNil(t, pb)
	assert.NotEmpty(t, pb)

	pbMsg, err := MessageProtoFromBytes(pb)
	assert.NoError(t, err)
	assert.NotNil(t, pbMsg)
	assert.Equal(t, m.Group, pbMsg.Group)
	assert.Equal(t, m.ID, pbMsg.Id)
	assert.Equal(t, m.Priority, pbMsg.Priority)
	assert.Equal(t, m.Content, pbMsg.Content)
	assert.Equal(t, m.Metadata, pbMsg.Metadata)
}

func TestMessage_ToBytesAndFromBytes_Roundtrip(t *testing.T) {
	orig := &Message{
		Group:    "grp",
		ID:       1001,
		Priority: -5,
		Content:  "payload",
		Metadata: map[string]string{
			"a": "1",
			"b": "2",
		},
	}

	b, err := orig.ToBytes()
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotEmpty(t, b)

	restored, err := MessageFromBytes(b)
	assert.NoError(t, err)
	assert.NotNil(t, restored)

	assert.Equal(t, orig.Group, restored.Group)
	assert.Equal(t, orig.ID, restored.ID)
	assert.Equal(t, orig.Priority, restored.Priority)
	assert.Equal(t, orig.Content, restored.Content)
	assert.Equal(t, orig.Metadata, restored.Metadata)
}

func TestMessage_UpdatePriority(t *testing.T) {
	m := &Message{Priority: 1}
	m.UpdatePriority(999)
	assert.Equal(t, int64(999), m.Priority)
}
