package weightedavl

import (
	"sync"
)

type node struct {
	key    string
	weight int
	sum    int
	height int
	left   *node
	right  *node
}

func height(n *node) int {
	if n == nil {
		return 0
	}
	return n.height
}

func sum(n *node) int {
	if n == nil {
		return 0
	}
	return n.sum
}

func update(n *node) {
	if n == nil {
		return
	}
	n.height = 1 + max(height(n.left), height(n.right))
	n.sum = n.weight + sum(n.left) + sum(n.right)
}

func rotateRight(y *node) *node {
	x := y.left
	T2 := x.right

	x.right = y
	y.left = T2

	update(y)
	update(x)

	return x
}

func rotateLeft(x *node) *node {
	y := x.right
	T2 := y.left

	y.left = x
	x.right = T2

	update(x)
	update(y)

	return y
}

func getBalance(n *node) int {
	if n == nil {
		return 0
	}
	return height(n.left) - height(n.right)
}

func insert(n *node, key string, weight int) *node {
	if n == nil {
		return &node{key: key, weight: weight, sum: weight, height: 1}
	}
	if key < n.key {
		n.left = insert(n.left, key, weight)
	} else if key > n.key {
		n.right = insert(n.right, key, weight)
	} else {
		n.weight = weight
		update(n)
		return n
	}

	update(n)
	balance := getBalance(n)

	// Left Left
	if balance > 1 && key < n.left.key {
		return rotateRight(n)
	}
	// Right Right
	if balance < -1 && key > n.right.key {
		return rotateLeft(n)
	}
	// Left Right
	if balance > 1 && key > n.left.key {
		n.left = rotateLeft(n.left)
		return rotateRight(n)
	}
	// Right Left
	if balance < -1 && key < n.right.key {
		n.right = rotateRight(n.right)
		return rotateLeft(n)
	}

	return n
}

func minValueNode(n *node) *node {
	curr := n
	for curr.left != nil {
		curr = curr.left
	}
	return curr
}

func delete(n *node, key string) *node {
	if n == nil {
		return nil
	}
	if key < n.key {
		n.left = delete(n.left, key)
	} else if key > n.key {
		n.right = delete(n.right, key)
	} else {
		if n.left == nil || n.right == nil {
			var temp *node
			if n.left != nil {
				temp = n.left
			} else {
				temp = n.right
			}
			return temp
		}
		temp := minValueNode(n.right)
		n.key = temp.key
		n.weight = temp.weight
		n.right = delete(n.right, temp.key)
	}

	update(n)
	balance := getBalance(n)

	if balance > 1 && getBalance(n.left) >= 0 {
		return rotateRight(n)
	}
	if balance > 1 && getBalance(n.left) < 0 {
		n.left = rotateLeft(n.left)
		return rotateRight(n)
	}
	if balance < -1 && getBalance(n.right) <= 0 {
		return rotateLeft(n)
	}
	if balance < -1 && getBalance(n.right) > 0 {
		n.right = rotateRight(n.right)
		return rotateLeft(n)
	}

	return n
}

func sample(n *node, r int) string {
	if n == nil {
		return ""
	}
	leftSum := sum(n.left)
	if r < leftSum {
		return sample(n.left, r)
	} else if r < leftSum+n.weight {
		return n.key
	} else {
		return sample(n.right, r-leftSum-n.weight)
	}
}

func get(n *node, key string) (int, bool) {
	if n == nil {
		return 0, false
	}
	if key < n.key {
		return get(n.left, key)
	} else if key > n.key {
		return get(n.right, key)
	} else {
		return n.weight, true
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type WeightedAVL struct {
	root *node
	mu   sync.RWMutex
}

func NewWeightedAVL() *WeightedAVL {
	return &WeightedAVL{
		mu: sync.RWMutex{},
	}
}

func (tree *WeightedAVL) Update(key string, weight int) {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	tree.root = insert(tree.root, key, weight)
}

func (tree *WeightedAVL) Remove(key string) {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	tree.root = delete(tree.root, key)
}

func (tree *WeightedAVL) Get(key string) (int, bool) {
	tree.mu.RLock()
	defer tree.mu.RUnlock()
	return get(tree.root, key)
}

// Sample picks a key with probability proportional to its weight. The pick is a
// pure function of seed and the tree's contents: the same seed on the same tree
// always returns the same key. Under Raft the seed comes from the log entry, so
// every replica, and every replay of the log, picks the same key.
func (tree *WeightedAVL) Sample(seed uint64) string {
	tree.mu.RLock()
	defer tree.mu.RUnlock()

	if tree.root == nil || tree.root.sum == 0 {
		return ""
	}

	r := int(mix64(seed) % uint64(tree.root.sum))
	return sample(tree.root, r)
}

// mix64 is the splitmix64 finalizer. It spreads consecutive seeds (such as
// consecutive Raft log indexes) uniformly over uint64, so their picks are not
// correlated.
func mix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

func (tree *WeightedAVL) Increment(key string, delta int) {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	weight, exists := get(tree.root, key)
	if !exists {
		weight = 0
	}
	tree.root = insert(tree.root, key, weight+delta)
}

func (tree *WeightedAVL) Sum() int {
	tree.mu.RLock()
	defer tree.mu.RUnlock()

	if tree.root == nil {
		return 0
	}
	return tree.root.sum
}

func (tree *WeightedAVL) Items() map[string]int {
	tree.mu.RLock()
	defer tree.mu.RUnlock()

	items := make(map[string]int)
	var traverse func(n *node)
	traverse = func(n *node) {
		if n != nil {
			traverse(n.left)
			items[n.key] = n.weight
			traverse(n.right)
		}
	}
	traverse(tree.root)
	return items
}
