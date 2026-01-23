package sstable

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"kvschool/internal/bloom"
)

// ErrNotImplemented используется в заготовке практики второго дня.
var ErrNotImplemented = errors.New("sstable: функция не реализована")
var ErrCorrupted = errors.New("sstable: файл поврежден")

type indexEntry struct {
	Key    []byte
	Offset int64
}

// Writer пишет отсортированные пары key/value (CDR) в файл.
// Формат файла должен позволять чтение без загрузки всего файла в память.
// Обычно это: [Data Block 1] [Data Block 2] ... [Sparse Index] [Footer].
type Writer struct {
	w           *bufio.Writer
	f           io.Writer
	offset      int64
	index       []indexEntry
	indexStep   int
	lastIndexAt int64
	bloom       *bloom.Filter
}

func NewWriter(w io.Writer) *Writer {
	bf := bloom.NewWithEstimates(10000, 0.1)

	return &Writer{
		w:         bufio.NewWriter(w),
		f:         w,
		indexStep: 1024,
		bloom:     bf,
	}
}

// Add добавляет пару. Ключи должны быть строго возрастающими.
func (w *Writer) Add(key, value []byte) error {
	if err := w.bloom.Add(key); err != nil {
		return err
	}

	if w.offset == 0 || (w.offset-w.lastIndexAt) > int64(w.indexStep) {
		w.index = append(w.index, indexEntry{
			Key:    append([]byte(nil), key...),
			Offset: w.offset,
		})
		w.lastIndexAt = w.offset
	}

	var lenBuf [4]byte
	n := 0

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(key)))
	k, err := w.w.Write(lenBuf[:])
	if err != nil {
		return err
	}
	n += k
	k, err = w.w.Write(key)
	if err != nil {
		return err
	}
	n += k

	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(value)))
	k, err = w.w.Write(lenBuf[:])
	if err != nil {
		return err
	}
	n += k
	k, err = w.w.Write(value)
	if err != nil {
		return err
	}
	n += k

	w.offset += int64(n)
	return nil
}

func (w *Writer) Close() error {
	indexOffset := w.offset

	var buf [8]byte
	binary.BigEndian.PutUint32(buf[:4], uint32(len(w.index)))
	if _, err := w.w.Write(buf[:4]); err != nil {
		return err
	}

	for _, entry := range w.index {
		binary.BigEndian.PutUint32(buf[:4], uint32(len(entry.Key)))
		if _, err := w.w.Write(buf[:4]); err != nil {
			return err
		}
		if _, err := w.w.Write(entry.Key); err != nil {
			return err
		}
		binary.BigEndian.PutUint64(buf[:], uint64(entry.Offset))
		if _, err := w.w.Write(buf[:]); err != nil {
			return err
		}
	}

	indexSize := 4
	for _, entry := range w.index {
		indexSize += 4 + len(entry.Key) + 8
	}
	bloomOffset := indexOffset + int64(indexSize)

	_, err := w.bloom.WriteTo(w.w)
	if err != nil {
		return err
	}

	binary.BigEndian.PutUint64(buf[:], uint64(indexOffset))
	if _, err := w.w.Write(buf[:]); err != nil {
		return err
	}

	binary.BigEndian.PutUint64(buf[:], uint64(bloomOffset))
	if _, err := w.w.Write(buf[:]); err != nil {
		return err
	}

	return w.w.Flush()
}

// Reader читает SSTable с диска.
// Использует RandomAccess (io.ReaderAt) для прыжков по индексу.
type Reader struct {
	r     io.ReaderAt
	size  int64
	index []indexEntry
}

func NewReader(r io.ReaderAt, size int64) (*Reader, error) {
	reader := &Reader{r: r, size: size}
	if err := reader.loadIndex(); err != nil {
		return nil, err
	}
	return reader, nil
}

func (r *Reader) loadIndex() error {
	if r.size < 8 {
		return errors.New("file too small")
	}

	var buf [8]byte
	if _, err := r.r.ReadAt(buf[:], r.size-8); err != nil {
		return err
	}
	indexOffset := int64(binary.BigEndian.Uint64(buf[:]))

	sectionSize := (r.size - 8) - indexOffset
	if sectionSize < 0 {
		return ErrCorrupted
	}

	indexData := make([]byte, sectionSize)
	if _, err := r.r.ReadAt(indexData, indexOffset); err != nil {
		return err
	}

	if len(indexData) < 4 {
		return ErrCorrupted
	}
	count := binary.BigEndian.Uint32(indexData[:4])
	r.index = make([]indexEntry, 0, count)

	pos := 4
	for i := 0; i < int(count); i++ {
		if pos+4 > len(indexData) {
			return ErrCorrupted
		}
		keyLen := int(binary.BigEndian.Uint32(indexData[pos : pos+4]))
		pos += 4

		if pos+keyLen+8 > len(indexData) {
			return ErrCorrupted
		}
		key := indexData[pos : pos+keyLen]
		pos += keyLen

		offset := int64(binary.BigEndian.Uint64(indexData[pos : pos+8]))
		pos += 8

		r.index = append(r.index, indexEntry{Key: key, Offset: offset})
	}
	return nil
}

// Iterator возвращает упорядоченную итерацию по диапазону [start, end).
// Использует Sparse Index, чтобы найти нужный блок данных.
func (r *Reader) Iterator(start, end []byte) (*Iter, error) {

	startOffset := int64(0)
	for _, entry := range r.index {
		if bytes.Compare(entry.Key, start) <= 0 {
			startOffset = entry.Offset
		} else {
			break
		}
	}

	sr := io.NewSectionReader(r.r, startOffset, r.size-8-startOffset)
	return &Iter{
		br:    bufio.NewReader(sr),
		end:   end,
		start: start,
	}, nil
}

type Iter struct {
	br    *bufio.Reader
	end   []byte
	start []byte
}

func (it *Iter) Next() (key, value []byte, ok bool, err error) {
	for {

		var lenBuf [4]byte
		_, err := io.ReadFull(it.br, lenBuf[:])
		if err == io.EOF {
			return nil, nil, false, nil
		}
		if err != nil {
			return nil, nil, false, err
		}
		keyLen := binary.BigEndian.Uint32(lenBuf[:])

		key = make([]byte, keyLen)
		if _, err := io.ReadFull(it.br, key); err != nil {
			return nil, nil, false, err
		}

		if _, err := io.ReadFull(it.br, lenBuf[:]); err != nil {
			return nil, nil, false, err
		}
		valLen := binary.BigEndian.Uint32(lenBuf[:])

		value = make([]byte, valLen)
		if _, err := io.ReadFull(it.br, value); err != nil {
			return nil, nil, false, err
		}

		if it.start != nil && bytes.Compare(key, it.start) < 0 {
			continue
		}

		if it.end != nil && bytes.Compare(key, it.end) >= 0 {
			return nil, nil, false, nil
		}

		return key, value, true, nil
	}
}

func (it *Iter) Close() error { return nil }
