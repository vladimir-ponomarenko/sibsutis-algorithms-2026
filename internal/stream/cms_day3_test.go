//go:build day3

package stream

import (
	"testing"
)

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

// func BenchmarkBloom_Negative(b *testing.B) {
// 	f := New(10000, 5)
// 	key := []byte("missing_key")

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		_, _ = f.MayContain(key)
// 	}
// }

// func TestBloom_FalsePositiveRate(t *testing.T) {
// 	n := 1000
// 	fpTarget := 0.05
// 	f := NewWithEstimates(uint64(n), fpTarget)

// 	for i := 0; i < n; i++ {
// 		f.Add([]byte(fmt.Sprintf("%d", i)))
// 	}

// 	fpCount := 0
// 	checkCount := 10000
// 	for i := 0; i < checkCount; i++ {
// 		ok, _ := f.MayContain([]byte(fmt.Sprintf("missing_%d", i)))
// 		if ok {
// 			fpCount++
// 		}
// 	}

// 	rate := float64(fpCount) / float64(checkCount)
// 	t.Logf("Target FP: %.4f, Actual: %.4f", fpTarget, rate)

// 	if rate > fpTarget*2 {
// 		t.Errorf("FP rate too high")
// 	}
// }
