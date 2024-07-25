package queue

type Queue interface {
	Enqueue(priority int, content string) (*Message, error)
	Dequeue() (*Message, error)
	GetByID(id uint64) (*Message, error)
	UpdatePriority(id uint64, newPriority int) error
	Len() int
}
