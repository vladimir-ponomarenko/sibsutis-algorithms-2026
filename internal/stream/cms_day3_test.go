//go:build day3

package stream

import "testing"

func TestCountMinSketch_EstimateMonotone(t *testing.T) {
	cms := NewCountMinSketch(64, 4, 1)

	for i := 0; i < 10; i++ {
		if err := cms.Add([]byte("hot")); err != nil {
			t.Fatalf("Add hot: %v", err)
		}
	}
	est, err := cms.Estimate([]byte("hot"))
	if err != nil {
		t.Fatalf("Estimate hot: %v", err)
	}
	// Для CMS типично: оценка >= истинного значения (overestimate допустим),
	// но undercount — индикатор ошибки.
	if est < 10 {
		t.Fatalf("estimate too small: %d", est)
	}
}
