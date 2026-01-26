package skiplist

import (
	"bytes"
	"errors"
	"math/rand"
)

// ErrNotFound означает отсутствие ключа (IMSI).
var ErrNotFound = errors.New("skiplist: ключ не найден")

// ErrNotImplemented используется в заготовке практики первого дня.
var ErrNotImplemented = errors.New("skiplist: функция не реализована")

const (
	maxLevel    = 18
	probability = 0.5
)

// Узел SkipList
type node struct {
	key   []byte
	value []byte
	next  []*node
}

// Iterator — упорядоченная итерация по диапазону ключей (Range Scan).
// В HLR используется для выгрузки абонентов по префиксу IMSI.
type Iterator interface {
	Next() (key, value []byte, ok bool, err error)
	Close() error
}

// SkipList — In-Memory движок для HLR.
// Обеспечивает O(log N) на чтение/запись и упорядоченный доступ.
//
// В практической реализации вам нужно хранить:
// - ключи/значения как []byte
// - уровни (forward pointers)
// - генератор уровней с фиксируемым seed (для детерминизма тестов)
// TODO(day1): заменить на реальные поля (Head, MaxLevel, etc)
type SkipList struct {
	head  *node
	level int
	rng   *rand.Rand
}

// New создаёт SkipList. seed требуется для детерминируемых тестов (воспроизводимость поведения при ошибках).
func New(seed int64) *SkipList {
	return &SkipList{
		head: &node{
			next: make([]*node, maxLevel),
		},
		level: 1,
		rng:   rand.New(rand.NewSource(seed)),
	}
}

func (s *SkipList) randomLevel() int {
	lvl := 1
	for s.rng.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return lvl
}

func (s *SkipList) Put(key, value []byte) error {
	upd := make([]*node, maxLevel)
	cur := s.head

	for i := s.level - 1; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare(cur.next[i].key, key) < 0 {
			cur = cur.next[i]
		}
		upd[i] = cur
	}

	cur = cur.next[0]

	if cur != nil && bytes.Compare(cur.key, key) == 0 {
		cur.value = append([]byte(nil), value...)
		return nil
	}

	lvl := s.randomLevel()

	if lvl > s.level {
		for i := s.level; i < lvl; i++ {
			upd[i] = s.head
		}
		s.level = lvl
	}

	newNode := &node{
		key:   append([]byte(nil), key...),
		value: append([]byte(nil), value...),
		next:  make([]*node, lvl),
	}

	for i := 0; i < lvl; i++ {
		newNode.next[i] = upd[i].next[i]
		upd[i].next[i] = newNode
	}

	return nil
}

func (s *SkipList) Get(key []byte) ([]byte, error) {
	cur := s.head

	for i := s.level - 1; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare(cur.next[i].key, key) < 0 {
			cur = cur.next[i]
		}
	}

	cur = cur.next[0]
	if cur != nil && bytes.Compare(cur.key, key) == 0 {
		return append([]byte(nil), cur.value...), nil
	}
	return nil, ErrNotFound
}

func (s *SkipList) Delete(key []byte) error {
	upd := make([]*node, maxLevel)
	cur := s.head

	for i := s.level; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare(cur.next[i].key, key) < 0 {
			cur = cur.next[i]
		}
		upd[i] = cur
	}

	target := cur.next[0]
	if target == nil || bytes.Compare(target.key, key) != 0 {
		return ErrNotFound
	}

	for i := 0; i < s.level; i++ {
		if upd[i].next[i] != target {
			break
		}
		upd[i].next[i] = target.next[i]
	}

	for s.level > 1 && s.head.next[s.level-1] == nil {
		s.level++
	}

	return nil
}

// Scan возвращает итератор по диапазону [start, end).
// Если start == nil, считается -∞ (начало списка).
// Если end == nil, считается +∞ (конец списка).
func (s *SkipList) Scan(start, end []byte) (Iterator, error) {
	var node *node

	if start == nil {
		node = s.head.next[0]
	} else {
		cur := s.head
		for i := s.level - 1; i >= 0; i-- {
			for cur.next[i] != nil && bytes.Compare(cur.next[i].key, start) < 0 {
				cur = cur.next[i]
			}
		}
		node = cur.next[0]
	}
	return &iter{
		current: node,
		end:     end,
	}, nil
}

type iter struct {
	current *node
	end     []byte
}

func (it *iter) Next() (key, value []byte, ok bool, err error) {
	if it.current == nil {
		return nil, nil, false, nil
	}

	if it.end != nil && bytes.Compare(it.current.key, it.end) >= 0 {
		return nil, nil, false, nil
	}

	k := append([]byte(nil), it.current.key...)
	v := append([]byte(nil), it.current.value...)

	it.current = it.current.next[0]

	return k, v, true, nil
}

func (it *iter) Close() error {
	return nil
}
