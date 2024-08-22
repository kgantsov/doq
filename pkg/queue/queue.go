package queue

type Queue interface {
	Enqueue(group string, item *Item)
	Dequeue() *Item
	GetByID(group string, id uint64) *Item
	DeleteByID(group string, id uint64) *Item
	UpdatePriority(group string, id uint64, priority int64)
	Len() uint64
}
