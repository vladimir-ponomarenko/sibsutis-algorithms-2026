//go:build day1

package skiplist

import (
	"bytes"
	"testing"
)

func TestSkipList_BasicCRUD(t *testing.T) {
	sl := New(1)

	if err := sl.Put([]byte("b"), []byte("2")); err != nil {
		t.Fatalf("Put b: %v", err)
	}
	if err := sl.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatalf("Put a: %v", err)
	}
	if err := sl.Put([]byte("c"), []byte("3")); err != nil {
		t.Fatalf("Put c: %v", err)
	}

	v, err := sl.Get([]byte("a"))
	if err != nil {
		t.Fatalf("Get a: %v", err)
	}
	if !bytes.Equal(v, []byte("1")) {
		t.Fatalf("Get a mismatch: %q", string(v))
	}

	if err := sl.Delete([]byte("b")); err != nil {
		t.Fatalf("Delete b: %v", err)
	}
	_, err = sl.Get([]byte("b"))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSkipList_ScanOrderAndRange(t *testing.T) {
	sl := New(1)
	_ = sl.Put([]byte("a"), []byte("1"))
	_ = sl.Put([]byte("b"), []byte("2"))
	_ = sl.Put([]byte("c"), []byte("3"))

	it, err := sl.Scan([]byte("b"), []byte("d"))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	defer it.Close()

	var keys []string
	for {
		k, _, ok, err := it.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if !ok {
			break
		}
		keys = append(keys, string(k))
	}
	if len(keys) != 2 || keys[0] != "b" || keys[1] != "c" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

// func TestSkipList_CornerCases(t *testing.T) {
// 	s1 := New(1)

// 	emptyKey := []byte{}
// 	val := []byte{"empty"}

// 	if err := sl.Put(emptyKey, val); err != nil {
// 		t.Fatalf("Put empty key: %v", err)
// 	}
// 	got, err := sl.Get(emptyKey)
// 	if err != nil || !bytes.Equal(got, val) {
// 		t.Errorf("Get empty key failed")
// 	}

// 	key := []byte("k")
// 	emptyVal := []byte{}
// 	if err := sl.Put(key, emptyVal); err != nil {
// 		t.Fatalf("Put empty val: %v", err)
// 	}
// 	got, err = sl.Get(key)
// 	if err != nil {
// 		t.Fatalf("Get empty val error: %v", err)
// 	}
// 	if got == nil || len(got) != 0 {
// 		t.Errorf("Get empty val mismatch: got %v", got)
// 	}
// }
