package memory

type LinkedListNode struct {
	group string
	queue *PriorityMemoryQueue
	prev  *LinkedListNode
	next  *LinkedListNode
}

func NewLinkedListNode(group string, queue *PriorityMemoryQueue) *LinkedListNode {
	node := new(LinkedListNode)
	node.group = group
	node.queue = queue

	return node
}

func (n *LinkedListNode) Queue() *PriorityMemoryQueue {
	return n.queue
}

type LinkedList struct {
	head  *LinkedListNode
	tail  *LinkedListNode
	total uint64
}

func (l *LinkedList) Append(node *LinkedListNode) {
	if l.head == nil {
		l.head = node
		l.tail = l.head
	} else {
		l.tail.next = node
		node.prev = l.tail
		l.tail = l.tail.next
	}

	l.total++
}

func (l *LinkedList) Remove(node *LinkedListNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}

	node.prev = nil
	node.next = nil
	node.queue = nil

	l.total--
}

func (l *LinkedList) Len() uint64 {
	return l.total
}
