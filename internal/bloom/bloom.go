package bloom

import (
	"encoding/binary"
	"errors"
	"hash/fnv"
	"io"
	"math"
)

// ErrNotImplemented используется в заготовке практики третьего дня.
var ErrNotImplemented = errors.New("bloom: функция не реализована")

// Filter — вероятностный фильтр Блума ("Охранник диска").
// Позволяет мгновенно сказать "НЕТ, ключа здесь нет" с вероятностью 100%.
// Если говорит "ВОЗМОЖНО ЕСТЬ", придется проверять диск.
type Filter struct {
	bitset []uint64
	m      uint64
	k      uint8
}

// New создает новый фильтр.
// size (m) — размер битового массива.
// hashes (k) — количество хеш-функций.
func New(size uint64, hashes uint8) *Filter {
	words := (size + 63) / 64

	return &Filter{
		bitset: make([]uint64, words),
		m:      words * 64,
		k:      hashes,
	}
}

// NewWithEstimates создает фильтр, вычисляя оптимальные m и k.
// n - ожидаемое кол-во элементов, fp - желаемая вероятность ложноположительных.
func NewWithEstimates(n uint64, fp float64) *Filter {
	m := float64(n) * math.Log(fp) / -math.Pow(math.Log(2), 2)
	k := (m / float64(n)) * math.Log(2)

	return New(uint64(math.Ceil(m)), uint8(math.Ceil(k)))
}

// Add добавляет ключ в фильтр.
func (f *Filter) Add(key []byte) error {
	h1, h2 := doubleHash(key)
	for i := uint8(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.m
		f.bitset[idx/64] |= (1 << (idx % 64))
	}
	return nil
}

// MayContain проверяет наличие ключа.
// Возвращает false, если ключа точно нет.
// Возвращает true, если ключ возможно есть (или произошел false positive).
func (f *Filter) MayContain(key []byte) (bool, error) {
	h1, h2 := doubleHash(key)
	for i := uint8(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.m
		if (f.bitset[idx/64] & (1 << (idx % 64))) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func doubleHash(key []byte) (uint64, uint64) {
	h := fnv.New64a()
	h.Write(key)
	sum1 := h.Sum64()
	sum2 := (sum1 >> 32) | (sum1 << 32)

	return sum1, sum2
}

func (f *Filter) WriteTo(w io.Writer) (int64, error) {
	var written int64

	if err := binary.Write(w, binary.BigEndian, f.m); err != nil {
		return written, err
	}
	written += 8

	if err := binary.Write(w, binary.BigEndian, f.k); err != nil {
		return written, err
	}
	written += 1

	if err := binary.Write(w, binary.BigEndian, f.bitset); err != nil {
		return written, err
	}
	written += int64(len(f.bitset) * 8)

	return written, nil
}

func Decode(data []byte) (*Filter, error) {
	if len(data) < 9 {
		return nil, errors.New("bloom: data is too short")
	}
	m := binary.BigEndian.Uint64(data[0:8])
	k := data[8]

	bitaetBytes := data[9:]
	words := (m + 63) / 64
	if uint64(len(bitaetBytes)) < words*8 {
		return nil, errors.New("bloom: bitset data corrupted")
	}

	bitset := make([]uint64, words)
	for i := 0; i < int(words); i++ {
		bitset[i] = binary.BigEndian.Uint64(bitaetBytes[i*8 : (i+1)*8])
	}

	return &Filter{
		bitset: bitset,
		m:      m,
		k:      k,
	}, nil
}
