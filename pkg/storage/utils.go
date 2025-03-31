package storage

import "encoding/binary"

func addPrefix(prefix []byte, key []byte) []byte {
	return append(prefix, key...)
}

// Converts a uint to a byte slice
func uint64ToBytes(u uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, u)
	return buf
}
