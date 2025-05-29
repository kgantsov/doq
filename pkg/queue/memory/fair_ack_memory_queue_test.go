package memory

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateWeight(t *testing.T) {
	tests := []struct {
		unacked int
		want    int
	}{
		{0, 10},
		{1, 8},
		{2, 7},
		{3, 6},
		{4, 5},
		{5, 3},
		{7, 1},
		{8, 0},
		{9, 0},
		{10, 0},
	}

	for _, tt := range tests {
		t.Run("unacked="+strconv.Itoa(tt.unacked), func(t *testing.T) {
			got := CalculateWeight(8, tt.unacked)
			assert.Equal(t, tt.want, got, "CalculateWeight(%d) = %d; want %d", tt.unacked, got, tt.want)
		})
	}
}
