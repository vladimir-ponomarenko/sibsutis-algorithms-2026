//go:build day3

package stream

import (
	"fmt"
	"math/rand"
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

func TestCountMinSketch_Accuracy(t *testing.T) {
	cms := NewCMSWithEstimates(0.01, 0.01)

	zipf := rand.NewZipf(rand.New(rand.NewSource(1)), 1.1, 1, 1000)
	freqs := make(map[string]int)
	total := 100000

	for i := 0; i < total; i++ {
		k := fmt.Sprintf("k-%d", zipf.Uint64())
		freqs[k]++
		cms.Add([]byte(k))
	}

	for k, realCount := range freqs {
		if realCount > total/100 {
			est, _ := cms.Estimate([]byte(k))
			errRate := float64(est-uint64(realCount)) / float64(total)

			if errRate > 0.02 {
				t.Errorf("Key %s: Real %d, Est %d, ErrorRate %.4f", k, realCount, est, errRate)
			}
		}
	}
}

func BenchmarkCMS_Add(b *testing.B) {
	cms := NewCountMinSketch(2000, 5)
	key := []byte("bench_key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cms.Add(key)
	}
}
