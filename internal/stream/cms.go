package stream

import "errors"

// ErrNotImplemented используется в заготовке практики третьего дня.
var ErrNotImplemented = errors.New("stream: функция не реализована")

// CountMinSketch — структура для поиска Top-Talkers (частых элементов).
// Использует фиксированный объем памяти (w * d счетчиков), чтобы считать трафик миллионов абонентов.
type CountMinSketch struct{}

// NewCountMinSketch создает скетч.
// width (w) — ширина таблицы (больше ширина -> меньше коллизий).
// depth (d) — количество хеш-функций (больше глубина -> выше точность).
func NewCountMinSketch(width, depth uint32) *CountMinSketch { return &CountMinSketch{} }

// Add увеличивает счетчик для ключа (например, +1 байт трафика).
func (c *CountMinSketch) Add(_ []byte) error {
	_ = c
	return ErrNotImplemented
}

// Estimate возвращает примерную частоту ключа.
// Гарантия: Estimate >= TrueCount (никогда не занижает).
func (c *CountMinSketch) Estimate(_ []byte) (uint64, error) {
	_ = c
	return 0, ErrNotImplemented
}
