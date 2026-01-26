package lsm

import "errors"

// ErrNotImplemented используется в заготовке практики второго дня.
var ErrNotImplemented = errors.New("lsm: функция не реализована")

// Options задаёт параметры LSM движка.
type Options struct {
	Dir string // Директория для хранения WAL и SSTables

	// Максимальный размер Memtable перед сбросом на диск (Flush).
	// В телекоме это баланс между памятью и частотой I/O.
	MemtableFlushThreshold int
}

// Engine — основной движок CDR Storage.
// Координирует работу Memtable, WAL и SSTables.
// Отвечает за Compaction (сборку мусора).
type Engine struct{}

func Open(_ Options) (*Engine, error) { return nil, ErrNotImplemented }

func (e *Engine) Close() error { return ErrNotImplemented }
