package skiplist

import (
	"encoding/binary"
	"testing"
)

func BenchmarkPut(b *testing.B) {
	s1 := New(40)
	b.ReportAllocs()
	b.ResetTimer()

	key := make([]byte, 8)
	val := []byte("payload")

	for i := 0; i < b.N; i++ {
		binary.BigEndian.PutUint64(key, uint64(i))
		_ = s1.Put(key, val)
	}
}

func BenchmarkGet(b *testing.B) {
	s1 := New(40)
	n := 100000

	for i := 0; i < n; i++ {
		key := make([]byte, 8)
		_ = s1.Put(key, []byte("date"))
	}

	b.ReportAllocs()
	b.ResetTimer()

	key := make([]byte, 8)
	for i := 0; i < b.N; i++ {
		k := i % n
		binary.BigEndian.PutUint64(key, uint64(k))
		_, _ = s1.Get(key)
	}
}
