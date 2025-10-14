package memory

type MemoryQueue interface {
	UpdateMaxUnacked(maxUnacked int) error
	Enqueue(group string, item *Item)
	Dequeue(ack bool) *Item
	Get(group string, id uint64) *Item
	Delete(group string, id uint64) *Item
	UpdatePriority(group string, id uint64, priority int64)
	Len() uint64
	UpdateWeights(group string, id uint64) error
}
