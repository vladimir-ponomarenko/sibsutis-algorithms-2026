# Day1: SkipList

## 1) Выбор параметров
maxLevel = 18 и p = 0.5
- При p=0.5
- maxLevel=18 - позволяет индексировать 2^18 элементов. При повышении этого числа, список продолжает работать, однако поиск начинает деградировать.

## 2) Эксперимент с p
1. p=0.1 - Редкий рост:
   - Память - минимальное использование.
   - Скорость - падает, слишком мало коротких путей.
2. p=0.9 - Частый рост:
   - Память - огромная, почти все узлы высотой maxLevel
   - Скорость - медленнее из-за постоянных аллокаций.
3. p=0.5 - Золотая середина

## 3) Безопасность
 В методах Put и Get реализовано глубокое копирование ключей и значений. Это добавит overhead, но это важно для безопасности данных внутри HLR.

## 4) Результаты
### Put:
``` bash
goos: linux
goarch: amd64
pkg: kvschool/internal/skiplist
cpu: AMD Ryzen 7 4700U with Radeon Graphics
=== RUN   BenchmarkPut
BenchmarkPut
BenchmarkPut-8           1928760               615.9 ns/op           112 B/op        4 allocs/op
PASS
ok      kvschool/internal/skiplist      1.864s
```

### Get
``` bash
goos: linux
goarch: amd64
pkg: kvschool/internal/skiplist
cpu: AMD Ryzen 7 4700U with Radeon Graphics
=== RUN   BenchmarkGet
BenchmarkGet
BenchmarkGet-8          60522904                19.97 ns/op            0 B/op        0 allocs/op
PASS
ok      kvschool/internal/skiplist      1.282s
```


go run ./cmd/kvtool load -count 100000 -store skiplist
go run ./cmd/kvtool wordcount -in ./testdata/text_small.txt -store skiplist