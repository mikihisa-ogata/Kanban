package domain

import (
	"math"
	"testing"
)

func TestValidStoryPoints(t *testing.T) {
	tests := []struct {
		points float64
		want   bool
	}{
		{0, true},
		{0.5, true},
		{3, true},
		{1.25, true},
		{999999.99, true},
		{1.255, false},
		{-1, false},
		{1000000, false},
		{math.NaN(), false},
		{math.Inf(1), false},
	}
	for _, tt := range tests {
		if got := ValidStoryPoints(tt.points); got != tt.want {
			t.Errorf("ValidStoryPoints(%v) = %v, want %v", tt.points, got, tt.want)
		}
	}
}
