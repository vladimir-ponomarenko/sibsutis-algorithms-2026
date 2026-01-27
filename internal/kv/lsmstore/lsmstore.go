package lsmstore

import (
	"context"
	"errors"

	"kvschool/internal/kv"
	"kvschool/internal/lsm"
)

// ErrNotImplemented используется в заготовке практики второго дня.
var ErrNotImplemented = errors.New("lsmstore: функция не реализована")

// Store — KV поверх LSM.
// В практической реализации вам нужно использовать пакеты internal/lsm, internal/sstable, internal/wal.
type Store struct {
	engine *lsm.Engine
}

type Options struct {
	Dir string
}

func Open(opts Options) (*Store, error) {
	e, err := lsm.Open(lsm.Options{
		Dir:                    opts.Dir,
		MemtableFlushThreshold: 1024 * 1024,
	})
	if err != nil {
		return nil, err
	}
	return &Store{engine: e}, nil
}

func (s *Store) Put(_ context.Context, key, value []byte) error {
	return s.engine.Put(key, value)
}

func (s *Store) Get(_ context.Context, key []byte) ([]byte, error) {
	v, err := s.engine.Get(key)
	if err != nil {
		return nil, kv.ErrNotFound
	}
	return v, nil
}

func (s *Store) Delete(_ context.Context, key []byte) error {
	return s.engine.Delete(key)
}

func (s *Store) Scan(_ context.Context, start, end []byte) (kv.Iterator, error) {
	it, err := s.engine.Scan(start, end)
	if err != nil {
		return nil, err
	}
	return &iterAdapter{it: it}, nil
}

func (s *Store) Close() error {
	return s.engine.Close()
}

type iterAdapter struct {
	it lsm.Iterator
}

func (i *iterAdapter) Next() (kv.Pair, bool, error) {
	k, v, ok, err := i.it.Next()
	if err != nil {
		return kv.Pair{}, false, err
	}
	if !ok {
		return kv.Pair{}, false, nil
	}
	return kv.Pair{Key: k, Value: v}, true, nil
}

func (i *iterAdapter) Close() error {
	return i.it.Close()
}

var _ kv.Store = (*Store)(nil)
