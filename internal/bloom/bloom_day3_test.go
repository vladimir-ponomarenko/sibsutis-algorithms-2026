//go:build day3

package bloom

import "testing"

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
