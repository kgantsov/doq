package raftstore

import (
	"bytes"
	"encoding/binary"

	"github.com/hashicorp/go-msgpack/v2/codec"
)

// Decode reverses the encode operation on a byte slice input
func DecodeMsgPack(buf []byte, out interface{}) error {
	r := bytes.NewBuffer(buf)
	hd := codec.MsgpackHandle{}
	dec := codec.NewDecoder(r, &hd)
	return dec.Decode(out)
}

// Encode writes an encoded object to a new bytes buffer
func EncodeMsgPack(in interface{}, useNewTimeFormat bool) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	hd := codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			TimeNotBuiltin: !useNewTimeFormat,
		},
	}
	enc := codec.NewEncoder(buf, &hd)
	err := enc.Encode(in)
	return buf, err
}

// Converts bytes to an integer
func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Converts a uint to a byte slice
func uint64ToBytes(u uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, u)
	return buf
}

func addPrefix(prefix []byte, key []byte) []byte {
	return append(prefix, key...)
}

// Copy the prefix into a new slice that is one larger than
// the prefix and add an `0xFF` byte to it so
func End(prefix []byte) []byte {
	end := make([]byte, len(prefix)+1)
	copy(end, prefix)

	end[len(end)-1] = 0xFF
	return end
}
