package queue

type Queue interface {
	Enqueue(group string, item *Item)
	Dequeue() *Item
	Get(group string, id uint64) *Item
	Delete(group string, id uint64) *Item
	UpdatePriority(group string, id uint64, priority int64)
	Len() uint64
}
