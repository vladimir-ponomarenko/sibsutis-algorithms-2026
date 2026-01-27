package lsm

import (
	"bytes"
	"errors"
	"fmt"
	"kvschool/internal/skiplist"
	"kvschool/internal/sstable"
	"kvschool/internal/wal"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

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
type Engine struct {
	opts    Options
	mu      sync.RWMutex
	mem     *skiplist.SkipList
	memSize int
	wal     *wal.Writer
	walFile *os.File
	ssts    []sstMetadata
}

type sstMetadata struct {
	filename string
	reader   *sstable.Reader
	created  time.Time
}

func Open(opts Options) (*Engine, error) {
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, err
	}

	e := &Engine{
		opts: opts,
		mem:  skiplist.New(time.Now().UnixNano()),
	}

	if err := e.loadSSTables(); err != nil {
		return nil, fmt.Errorf("load sst: %w", err)
	}

	if err := e.recoverFromWAL(); err != nil {
		return nil, fmt.Errorf("recover wal: %w", err)
	}

	if err := e.rotateWAL(); err != nil {
		return nil, fmt.Errorf("rotate wal: %w", err)
	}

	return e, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.wal != nil {
		return e.wal.Close()
	}
	return nil
}

func (e *Engine) Put(key, value []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rec := wal.Record{Type: wal.OpPut, Key: key, Value: value}
	if err := e.wal.Append(rec); err != nil {
		return err
	}

	if err := e.mem.Put(key, value); err != nil {
		return err
	}
	e.memSize += len(key) + len(value)

	if e.memSize >= e.opts.MemtableFlushThreshold {
		return e.flushLocked()
	}

	return nil
}

func (e *Engine) Delete(key []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rec := wal.Record{Type: wal.OpDelete, Key: key}
	if err := e.wal.Append(rec); err != nil {
		return err
	}

	return e.mem.Put(key, nil)
}

func (e *Engine) Get(key []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	v, err := e.mem.Get(key)
	if err == nil {
		if v == nil {
			return nil, errors.New("not found (deleted)")
		}
		return v, nil
	}

	for _, sst := range e.ssts {
		it, err := sst.reader.Iterator(key, nil)
		if err != nil {
			continue
		}
		k, val, ok, _ := it.Next()
		if ok && bytes.Equal(k, key) {
			if val == nil {
				return nil, errors.New("not found (deleted)")
			}
			return val, nil
		}
	}

	return nil, errors.New("not found")
}

func (e *Engine) Scan(start, end []byte) (Iterator, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var iters []Iterator
	memIter, err := e.mem.Scan(start, end)
	if err != nil {
		return nil, err
	}
	iters = append(iters, memIter)
	for _, sst := range e.ssts {
		sstIter, err := sst.reader.Iterator(start, end)
		if err != nil {
			return nil, err
		}
		iters = append(iters, sstIter)
	}
	mi := NewMergeIterator(iters)

	return &TombstoneFilterIterator{inner: mi}, nil
}

func (e *Engine) flushLocked() error {
	filename := fmt.Sprintf("%06d.sst", time.Now().UnixNano())
	path := filepath.Join(e.opts.Dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := sstable.NewWriter(f)
	it, _ := e.mem.Scan(nil, nil)
	for {
		k, v, ok, _ := it.Next()
		if !ok {
			break
		}
		if err := w.Add(k, v); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	rf, err := os.Open(path)
	if err != nil {
		return err
	}
	stat, _ := rf.Stat()
	reader, err := sstable.NewReader(rf, stat.Size())
	if err != nil {
		return err
	}

	newMeta := sstMetadata{filename: filename, reader: reader, created: time.Now()}
	e.ssts = append([]sstMetadata{newMeta}, e.ssts...)

	e.mem = skiplist.New(time.Now().UnixNano())
	e.memSize = 0

	if err := e.rotateWAL(); err != nil {
		return err
	}

	if len(e.ssts) > 3 {
		go e.Compact()
	}

	return nil
}

func (e *Engine) rotateWAL() error {
	if e.wal != nil {
		e.wal.Close()
	}
	filename := fmt.Sprintf("%06d.wal", time.Now().UnixNano())
	path := filepath.Join(e.opts.Dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	e.walFile = f
	e.wal = wal.NewWriter(f)
	return nil
}

func (e *Engine) loadSSTables() error {
	files, err := os.ReadDir(e.opts.Dir)
	if err != nil {
		return err
	}
	var ssts []sstMetadata
	for _, entry := range files {
		if filepath.Ext(entry.Name()) == ".sst" {
			path := filepath.Join(e.opts.Dir, entry.Name())
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			stat, _ := f.Stat()
			r, err := sstable.NewReader(f, stat.Size())
			if err != nil {
				return err
			}
			ssts = append(ssts, sstMetadata{filename: entry.Name(), reader: r})
		}
	}
	sort.Slice(ssts, func(i, j int) bool {
		return ssts[i].filename > ssts[j].filename
	})
	e.ssts = ssts
	return nil
}

func (e *Engine) recoverFromWAL() error {
	files, err := os.ReadDir(e.opts.Dir)
	if err != nil {
		return err
	}
	var walFiles []string
	for _, entry := range files {
		if filepath.Ext(entry.Name()) == ".wal" {
			walFiles = append(walFiles, filepath.Join(e.opts.Dir, entry.Name()))
		}
	}
	sort.Strings(walFiles)

	for _, path := range walFiles {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		r := wal.NewReader(f)
		for {
			rec, ok, _ := r.Next()
			if !ok {
				break
			}
			if rec.Type == wal.OpPut {
				e.mem.Put(rec.Key, rec.Value)
			} else if rec.Type == wal.OpDelete {
				e.mem.Put(rec.Key, nil)
			}
		}
		f.Close()
	}
	return nil
}
func (e *Engine) Compact() error {
	e.mu.Lock()
	toCompact := make([]sstMetadata, len(e.ssts))
	copy(toCompact, e.ssts)
	e.mu.Unlock()

	if len(toCompact) <= 1 {
		return nil
	}

	var iters []Iterator
	for _, meta := range toCompact {
		it, err := meta.reader.Iterator(nil, nil)
		if err != nil {
			return err
		}
		iters = append(iters, it)
	}

	mergeIter := NewMergeIterator(iters)

	newFilename := fmt.Sprintf("%06d.sst", time.Now().UnixNano())
	newPath := filepath.Join(e.opts.Dir, newFilename)
	f, err := os.Create(newPath)
	if err != nil {
		return err
	}
	w := sstable.NewWriter(f)

	for {
		k, v, ok, _ := mergeIter.Next()
		if !ok {
			break
		}
		if v != nil {
			if err := w.Add(k, v); err != nil {
				return err
			}
		}
	}
	w.Close()
	f.Close()

	rf, err := os.Open(newPath)
	if err != nil {
		return err
	}
	stat, _ := rf.Stat()
	r, err := sstable.NewReader(rf, stat.Size())
	if err != nil {
		return err
	}

	e.mu.Lock()
	compactedMap := make(map[string]bool)
	for _, m := range toCompact {
		compactedMap[m.filename] = true
	}
	var keptSSTs []sstMetadata
	for _, m := range e.ssts {
		if !compactedMap[m.filename] {
			keptSSTs = append(keptSSTs, m)
		}
	}
	keptSSTs = append(keptSSTs, sstMetadata{filename: newFilename, reader: r, created: time.Now()})
	e.ssts = keptSSTs
	e.mu.Unlock()

	for _, m := range toCompact {
		_ = os.Remove(filepath.Join(e.opts.Dir, m.filename))
	}

	return nil
}

type Iterator interface {
	Next() (key, value []byte, ok bool, err error)
	Close() error
}

type MergeIterator struct {
	iters []Iterator
	curr  []struct {
		k, v []byte
		ok   bool
		err  error
	}
	initialized bool
}

func NewMergeIterator(iters []Iterator) *MergeIterator {
	return &MergeIterator{
		iters: iters,
		curr: make([]struct {
			k, v []byte
			ok   bool
			err  error
		}, len(iters)),
	}
}

func (mi *MergeIterator) init() {
	for i, it := range mi.iters {
		mi.curr[i].k, mi.curr[i].v, mi.curr[i].ok, mi.curr[i].err = it.Next()
	}
	mi.initialized = true
}

func (mi *MergeIterator) Next() (key, value []byte, ok bool, err error) {
	if !mi.initialized {
		mi.init()
	}

	minIndex := -1
	var minKey []byte

	for i := 0; i < len(mi.iters); i++ {
		if !mi.curr[i].ok {
			continue
		}
		if mi.curr[i].err != nil {
			return nil, nil, false, mi.curr[i].err
		}
		if minIndex == -1 || bytes.Compare(mi.curr[i].k, minKey) < 0 {
			minIndex = i
			minKey = mi.curr[i].k
		}
	}

	if minIndex == -1 {
		return nil, nil, false, nil
	}

	resKey := minKey
	resVal := mi.curr[minIndex].v

	for i := 0; i < len(mi.iters); i++ {
		if mi.curr[i].ok && bytes.Equal(mi.curr[i].k, resKey) {
			mi.curr[i].k, mi.curr[i].v, mi.curr[i].ok, mi.curr[i].err = mi.iters[i].Next()
		}
	}

	return resKey, resVal, true, nil
}

func (mi *MergeIterator) Close() error {
	var firstErr error
	for _, it := range mi.iters {
		if err := it.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type TombstoneFilterIterator struct {
	inner Iterator
}

func (t *TombstoneFilterIterator) Next() (key, value []byte, ok bool, err error) {
	for {
		k, v, ok, err := t.inner.Next()
		if !ok || err != nil {
			return k, v, ok, err
		}
		if v == nil {
			continue
		}
		return k, v, true, nil
	}
}

func (t *TombstoneFilterIterator) Close() error {
	return t.inner.Close()
}
