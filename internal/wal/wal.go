package wal

import (
	"errors"
	"io"
)

// ErrNotImplemented используется в заготовке практики Дня 2.
var ErrNotImplemented = errors.New("wal: функция не реализована")

// OpType — тип операции в WAL (Put или Delete).
type OpType byte

const (
	OpPut    OpType = 1
	OpDelete OpType = 2
)

// Record — запись в логе.
// Используется для восстановления Memtable после сбоя (Crash Recovery).
type Record struct {
	Type  OpType
	Key   []byte
	Value []byte // только для Put
}

// Writer — append-only запись в лог.
// Гарантирует, что данные записаны до того, как мы подтвердим успешность операции пользователю.
type Writer struct{}

func NewWriter(_ io.Writer) *Writer { return &Writer{} }

func (w *Writer) Append(_ Record) error { return ErrNotImplemented }

func (w *Writer) Close() error { return ErrNotImplemented }

// Reader — последовательное чтение лога при старте системы.
type Reader struct{}

func NewReader(_ io.Reader) *Reader { return &Reader{} }

func (r *Reader) Next() (Record, bool, error) {
	return Record{}, false, ErrNotImplemented
}
