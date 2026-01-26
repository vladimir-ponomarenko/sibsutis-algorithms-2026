package sstable

import (
	"errors"
	"io"
)

// ErrNotImplemented используется в заготовке практики второго дня.
var ErrNotImplemented = errors.New("sstable: функция не реализована")

// Writer пишет отсортированные пары key/value (CDR) в файл.
// Формат файла должен позволять чтение без загрузки всего файла в память.
// Обычно это: [Data Block 1] [Data Block 2] ... [Sparse Index] [Footer].
type Writer struct{}

func NewWriter(_ io.Writer) *Writer { return &Writer{} }

// Add добавляет пару. Ключи должны быть строго возрастающими.
func (w *Writer) Add(_ []byte, _ []byte) error { return ErrNotImplemented }

func (w *Writer) Close() error { return ErrNotImplemented }

// Reader читает SSTable с диска.
// Использует RandomAccess (io.ReaderAt) для прыжков по индексу.
type Reader struct{}

func NewReader(_ io.ReaderAt, _ int64) *Reader { return &Reader{} }

// Iterator возвращает упорядоченную итерацию по диапазону [start, end).
// Использует Sparse Index, чтобы найти нужный блок данных.
func (r *Reader) Iterator(_ []byte, _ []byte) (*Iter, error) { return nil, ErrNotImplemented }

type Iter struct{}

func (it *Iter) Next() (key, value []byte, ok bool, err error) {
	return nil, nil, false, ErrNotImplemented
}

func (it *Iter) Close() error { return ErrNotImplemented }
