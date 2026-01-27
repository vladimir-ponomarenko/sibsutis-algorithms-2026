package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// ErrNotImplemented используется в заготовке практики Дня 2.
var ErrNotImplemented = errors.New("wal: функция не реализована")
var ErrCorrupted = errors.New("wal: файл повреждён")

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
type Writer struct {
	f *os.File
	w *bufio.Writer
}

func NewWriter(f *os.File) *Writer {
	return &Writer{
		f: f,
		w: bufio.NewWriter(f),
	}
}

func (w *Writer) Append(r Record) error {
	payloadLen := 1 + 4 + len(r.Key) + 4 + len(r.Value)
	totalLen := uint32(payloadLen)

	var lenBuf [4]byte

	binary.BigEndian.PutUint32(lenBuf[:], totalLen)
	if _, err := w.w.Write(lenBuf[:]); err != nil {
		return err
	}

	if err := w.w.WriteByte(byte(r.Type)); err != nil {
		return err
	}

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(r.Key)))
	if _, err := w.w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.w.Write(r.Key); err != nil {
		return err
	}

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(r.Value)))
	if _, err := w.w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.w.Write(r.Value); err != nil {
		return err
	}

	return w.w.Flush()
}

func (w *Writer) Close() error {
	if err := w.w.Flush(); err != nil {
		return err
	}
	return w.f.Close()
}

// Reader — последовательное чтение лога при старте системы.
type Reader struct {
	r io.Reader
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: r,
	}
}

func (r *Reader) Next() (Record, bool, error) {
	var lenBuf [4]byte

	_, err := io.ReadFull(r.r, lenBuf[:])
	if err == io.EOF {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	totalLen := binary.BigEndian.Uint32(lenBuf[:])

	payload := make([]byte, totalLen)
	if _, err := io.ReadFull(r.r, payload); err != nil {
		return Record{}, false, ErrCorrupted
	}

	op := OpType(payload[0])

	offset := 1
	if len(payload) < offset+4 {
		return Record{}, false, ErrCorrupted
	}
	keyLen := binary.BigEndian.Uint32(payload[offset : offset+4])
	offset += 4

	if uint32(len(payload)) < uint32(offset)+keyLen {
		return Record{}, false, ErrCorrupted
	}

	key := payload[offset : offset+int(keyLen)]
	offset += int(keyLen)

	if len(payload) < offset+4 {
		return Record{}, false, ErrCorrupted
	}

	valLen := binary.BigEndian.Uint32(payload[offset : offset+4])
	offset += 4

	if uint32(len(payload)) < uint32(offset)+valLen {
		return Record{}, false, ErrCorrupted
	}
	value := payload[offset : offset+int(valLen)]

	return Record{
		Type:  op,
		Key:   key,
		Value: value,
	}, true, nil
}
