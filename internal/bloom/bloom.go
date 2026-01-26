package bloom

import "errors"

// ErrNotImplemented используется в заготовке практики третьего дня.
var ErrNotImplemented = errors.New("bloom: функция не реализована")

// Filter — вероятностный фильтр Блума ("Охранник диска").
// Позволяет мгновенно сказать "НЕТ, ключа здесь нет" с вероятностью 100%.
// Если говорит "ВОЗМОЖНО ЕСТЬ", придется проверять диск.
type Filter struct{}

// New создает новый фильтр.
// size (m) — размер битового массива.
// hashes (k) — количество хеш-функций.
func New(size uint64, hashes uint8) *Filter { return &Filter{} }

// Add добавляет ключ в фильтр.
func (f *Filter) Add(_ []byte) error {
	_ = f
	return ErrNotImplemented
}

// MayContain проверяет наличие ключа.
// Возвращает false, если ключа точно нет.
// Возвращает true, если ключ возможно есть (или произошел false positive).
func (f *Filter) MayContain(_ []byte) (bool, error) {
	_ = f
	return false, ErrNotImplemented
}
