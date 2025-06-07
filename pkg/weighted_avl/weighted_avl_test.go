package weightedavl

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Set a fixed seed for deterministic test results
	rand.Seed(42)
}

func TestNewWeightedAVL(t *testing.T) {
	tree := NewWeightedAVL()
	assert.NotNil(t, tree)
	assert.Nil(t, tree.root)
	assert.Equal(t, 0, tree.Sum())
}

func TestUpdate(t *testing.T) {
	tree := NewWeightedAVL()

	// Test adding new nodes
	tree.Update("a", 5)
	assert.Equal(t, 5, tree.Sum())

	tree.Update("b", 3)
	assert.Equal(t, 8, tree.Sum())

	tree.Update("c", 7)
	assert.Equal(t, 15, tree.Sum())

	// Test updating existing node
	tree.Update("a", 10)
	assert.Equal(t, 20, tree.Sum())
}

func TestRemove(t *testing.T) {
	tree := NewWeightedAVL()

	// Test removing from empty tree
	tree.Remove("a")
	assert.Equal(t, 0, tree.Sum())

	// Add some nodes
	tree.Update("a", 5)
	tree.Update("b", 3)
	tree.Update("c", 7)

	// Test removing leaf node
	tree.Remove("b")
	assert.Equal(t, 12, tree.Sum())

	// Test removing node with one child
	tree.Remove("a")
	assert.Equal(t, 7, tree.Sum())

	// Test removing node with two children
	tree.Update("a", 5)
	tree.Update("b", 3)
	tree.Remove("a")
	assert.Equal(t, 10, tree.Sum())
}

func TestSample(t *testing.T) {
	tree := NewWeightedAVL()

	// Test sampling from empty tree
	assert.Equal(t, "", tree.Sample())

	// Test sampling from tree with one node
	tree.Update("a", 5)
	assert.Equal(t, "a", tree.Sample())

	// Test sampling with multiple nodes
	tree.Update("b", 3)
	tree.Update("c", 7)

	// Run multiple samples to verify distribution
	samples := make(map[string]int)
	for i := 0; i < 1000; i++ {
		sample := tree.Sample()
		samples[sample]++
	}

	// Verify that all keys are sampled
	assert.Contains(t, samples, "a")
	assert.Contains(t, samples, "b")
	assert.Contains(t, samples, "c")

	// Verify rough distribution (allowing for some variance due to randomness)
	total := samples["a"] + samples["b"] + samples["c"]
	assert.InDelta(t, float64(samples["a"])/float64(total), 0.33, 0.1)
	assert.InDelta(t, float64(samples["b"])/float64(total), 0.20, 0.1)
	assert.InDelta(t, float64(samples["c"])/float64(total), 0.47, 0.1)
}

func TestSum(t *testing.T) {
	tree := NewWeightedAVL()

	// Test empty tree
	assert.Equal(t, 0, tree.Sum())

	// Test tree with one node
	tree.Update("a", 5)
	assert.Equal(t, 5, tree.Sum())

	// Test tree with multiple nodes
	tree.Update("b", 3)
	tree.Update("c", 7)
	assert.Equal(t, 15, tree.Sum())

	// Test after removal
	tree.Remove("b")
	assert.Equal(t, 12, tree.Sum())
}

func TestAVLBalance(t *testing.T) {
	tree := NewWeightedAVL()

	// Test right rotation
	tree.Update("c", 1)
	tree.Update("b", 1)
	tree.Update("a", 1)
	assert.Equal(t, 3, tree.Sum())

	// Test left rotation
	tree = NewWeightedAVL()
	tree.Update("a", 1)
	tree.Update("b", 1)
	tree.Update("c", 1)
	assert.Equal(t, 3, tree.Sum())

	// Test left-right rotation
	tree = NewWeightedAVL()
	tree.Update("c", 1)
	tree.Update("a", 1)
	tree.Update("b", 1)
	assert.Equal(t, 3, tree.Sum())

	// Test right-left rotation
	tree = NewWeightedAVL()
	tree.Update("a", 1)
	tree.Update("c", 1)
	tree.Update("b", 1)
	assert.Equal(t, 3, tree.Sum())
}

func TestWeightedAVL_Get(t *testing.T) {
	tree := NewWeightedAVL()
	tree.Update("a", 10)
	tree.Update("b", 20)
	tree.Update("c", 30)

	weight, ok := tree.Get("a")
	if !ok || weight != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", weight, ok)
	}

	weight, ok = tree.Get("b")
	if !ok || weight != 20 {
		t.Errorf("expected (20, true), got (%d, %v)", weight, ok)
	}

	weight, ok = tree.Get("c")
	if !ok || weight != 30 {
		t.Errorf("expected (30, true), got (%d, %v)", weight, ok)
	}

	weight, ok = tree.Get("d")
	if ok || weight != 0 {
		t.Errorf("expected (0, false), got (%d, %v)", weight, ok)
	}

	tree.Remove("b")
	weight, ok = tree.Get("b")
	if ok || weight != 0 {
		t.Errorf("expected (0, false) after remove, got (%d, %v)", weight, ok)
	}
}

func TestWeightedAVLIncrement(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]int
		key      string
		delta    int
		expected int
	}{
		{
			name:     "increment new key",
			initial:  map[string]int{},
			key:      "test",
			delta:    5,
			expected: 5,
		},
		{
			name:     "increment existing key",
			initial:  map[string]int{"test": 10},
			key:      "test",
			delta:    5,
			expected: 15,
		},
		{
			name:     "decrement existing key",
			initial:  map[string]int{"test": 10},
			key:      "test",
			delta:    -3,
			expected: 7,
		},
		{
			name:     "decrement to zero",
			initial:  map[string]int{"test": 5},
			key:      "test",
			delta:    -5,
			expected: 0,
		},
		{
			name:     "decrement below zero",
			initial:  map[string]int{"test": 5},
			key:      "test",
			delta:    -7,
			expected: -2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewWeightedAVL()

			// Initialize tree with initial values
			for k, v := range tt.initial {
				tree.Update(k, v)
			}

			// Perform increment/decrement
			tree.Increment(tt.key, tt.delta)

			// Verify the result
			weight, exists := tree.Get(tt.key)
			if !exists {
				t.Errorf("key %s not found after increment", tt.key)
			}
			if weight != tt.expected {
				t.Errorf("expected weight %d, got %d", tt.expected, weight)
			}

			// Verify total sum
			expectedSum := 0
			for _, v := range tt.initial {
				expectedSum += v
			}
			expectedSum += tt.delta
			if tree.Sum() != expectedSum {
				t.Errorf("expected total sum %d, got %d", expectedSum, tree.Sum())
			}
		})
	}
}

func TestWeightedAVLIncrementConcurrent(t *testing.T) {
	tree := NewWeightedAVL()
	const numGoroutines = 10
	const numIncrements = 1000
	done := make(chan bool)

	// Spawn multiple goroutines that increment the same key
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIncrements; j++ {
				tree.Increment("test", 1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final weight
	weight, exists := tree.Get("test")
	if !exists {
		t.Error("key 'test' not found after concurrent increments")
	}
	expectedWeight := numGoroutines * numIncrements
	if weight != expectedWeight {
		t.Errorf("expected weight %d, got %d", expectedWeight, weight)
	}
}
