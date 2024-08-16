package queue

type Queue interface {
	Enqueue(priority int64, content string) (*Message, error)
	Dequeue(ack bool) (*Message, error)
	Ack(id uint64) error
	GetByID(id uint64) (*Message, error)
	UpdatePriority(id uint64, newPriority int64) error
	Len() int
}
