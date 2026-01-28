package stream

import (
	"errors"
	"hash/fnv"
	"math"
)

// ErrNotImplemented используется в заготовке практики третьего дня.
var ErrNotImplemented = errors.New("stream: функция не реализована")

// CountMinSketch — структура для поиска Top-Talkers (частых элементов).
// Использует фиксированный объем памяти (w * d счетчиков), чтобы считать трафик миллионов абонентов.
type CountMinSketch struct {
	counts []uint64
	width  uint32
	depth  uint32
	seed   uint64
}

// NewCountMinSketch создает скетч.
// width (w) — ширина таблицы (больше ширина -> меньше коллизий).
// depth (d) — количество хеш-функций (больше глубина -> выше точность).
func NewCountMinSketch(width, depth uint32, seeds ...int) *CountMinSketch {
	var s uint64 = 1
	if len(seeds) > 0 {
		s = uint64(seeds[0])
	}

	return &CountMinSketch{
		counts: make([]uint64, width*depth),
		width:  width,
		depth:  depth,
		seed:   s,
	}
}

func NewCMSWithEstimates(epsilon, delta float64) *CountMinSketch {
	width := uint32(math.Ceil(math.E / epsilon))
	depth := uint32(math.Ceil(math.Log(1.0 / delta)))

	return NewCountMinSketch(width, depth)
}

// Add увеличивает счетчик для ключа (например, +1 байт трафика).
func (c *CountMinSketch) Add(key []byte) error {
	h1, h2 := cmsHash(key, c.seed)

	for i := uint32(0); i < c.depth; i++ {
		// hash = h1 + i*h2
		h := h1 + uint64(i)*h2
		col := uint32(h % uint64(c.width))
		idx := i*c.width + col
		c.counts[idx]++
	}
	return nil
}

// Estimate возвращает примерную частоту ключа.
// Гарантия: Estimate >= TrueCount (никогда не занижает).
func (c *CountMinSketch) Estimate(key []byte) (uint64, error) {
	h1, h2 := cmsHash(key, c.seed)
	minCount := uint64(math.MaxUint64)

	for i := uint32(0); i < c.depth; i++ {
		h := h1 + uint64(i)*h2
		col := uint32(h % uint64(c.width))
		idx := i*c.width + col

		val := c.counts[idx]
		if val < minCount {
			minCount = val
		}
	}
	return minCount, nil
}

func cmsHash(key []byte, seed uint64) (uint64, uint64) {
	h := fnv.New64a()
	h.Write([]byte{byte(seed), byte(seed >> 8), byte(seed >> 16), byte(seed >> 24)})
	h.Write(key)
	sum1 := h.Sum64()
	sum2 := (sum1 >> 32) | (sum1 << 32)
	return sum1, sum2
}
