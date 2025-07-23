package storage

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	pb "github.com/kgantsov/doq/pkg/proto"
	"google.golang.org/protobuf/proto"
)

func WriteSnapshotItem(sink raft.SnapshotSink, item *pb.SnapshotItem) error {
	// Serialize the protobuf message
	data, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot item: %v", err)
	}

	// Write length prefix (4 bytes) followed by the data
	// This makes it easier to read back during restoration
	lengthBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBuf, uint32(len(data)))

	if _, err := sink.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to write length prefix: %v", err)
	}

	if _, err := sink.Write(data); err != nil {
		return fmt.Errorf("failed to write snapshot data: %v", err)
	}

	return nil
}

func ReadSnapshotItem(r io.Reader) (*pb.SnapshotItem, error) {
	lengthBuf := make([]byte, 4)
	n, err := io.ReadFull(r, lengthBuf)
	if err == io.EOF {
		return nil, err // End of snapshot
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read length prefix: %v", err)
	}
	if n != 4 {
		return nil, fmt.Errorf("incomplete length prefix read")
	}

	// Get the data length
	dataLength := binary.LittleEndian.Uint32(lengthBuf)

	// Read the protobuf data
	data := make([]byte, dataLength)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("failed to read snapshot data: %v", err)
	}

	// Unmarshal the SnapshotItem
	var item pb.SnapshotItem
	if err := proto.Unmarshal(data, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot item: %v", err)
	}

	return &item, nil
}

func ReadSnapshotQueueItem(r io.Reader) (*pb.Queue, error) {
	item, err := ReadSnapshotItem(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot item: %v", err)
	}

	switch v := item.Item.(type) {
	case *pb.SnapshotItem_Queue:
		return v.Queue, nil
	default:
		return nil, fmt.Errorf("incorrect snapshot item type")
	}
}

func ReadSnapshotMessageItem(r io.Reader) (*pb.Message, error) {
	item, err := ReadSnapshotItem(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot item: %v", err)
	}

	switch v := item.Item.(type) {
	case *pb.SnapshotItem_Message:
		return v.Message, nil
	default:
		return nil, fmt.Errorf("incorrect snapshot item type")
	}
}
