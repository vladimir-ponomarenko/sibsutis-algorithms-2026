//go:build day3

package bloom

import (
	"fmt"
	"testing"
)

func TestBloom_NoFalseNegatives(t *testing.T) {
	// Параметры маленькие намеренно: цель теста — свойство "нет false negative",
	// а не качество false positive.
	f := New(1024, 3)

	keys := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	for _, k := range keys {
		if err := f.Add(k); err != nil {
			t.Fatalf("Add(%q): %v", string(k), err)
		}
	}
	for _, k := range keys {
		ok, err := f.MayContain(k)
		if err != nil {
			t.Fatalf("MayContain(%q): %v", string(k), err)
		}
		if !ok {
			t.Fatalf("false negative for key=%q", string(k))
		}
	}
}

func TestBloom_FalsePositiveRate(t *testing.T) {
	n := 10000
	targetFP := 0.01

	f := NewWithEstimates(uint64(n), targetFP)

	for i := 0; i < n; i++ {
		f.Add([]byte(fmt.Sprintf("k_%d", i)))
	}

	missCount := 0
	checks := 100000
	for i := 0; i < checks; i++ {
		ok, _ := f.MayContain([]byte(fmt.Sprintf("missing_%d", i)))
		if ok {
			missCount++
		}
	}

	rate := float64(missCount) / float64(checks)
	t.Logf("Target FP: %.4f, Actual FP: %.4f", targetFP, rate)

	if rate > targetFP*2 {
		t.Errorf("FP Rate too high: got %.4f, want ~%.4f", rate, targetFP)
	}
}

func BenchmarkBloom_Guard(b *testing.B) {
	f := NewWithEstimates(10000, 0.01)
	key := []byte("missing_key")

	b.Run("BloomFilter_Check", func(b *testing.B) {
		for i := 0; i < b.N; i++ {

			_, _ = f.MayContain(key)
		}
	})

	b.Run("Disk_Seek_Simulation", func(b *testing.B) {

		mockDisk := make(map[string]bool)
		for i := 0; i < b.N; i++ {
			_ = mockDisk[string(key)]
		}
	})
}
