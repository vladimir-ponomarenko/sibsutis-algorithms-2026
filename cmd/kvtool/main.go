package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"kvschool/internal/kv"
	"kvschool/internal/kv/lsmstore"
	"kvschool/internal/kv/memmap"
	"kvschool/internal/mapreduce"
	"kvschool/internal/stream"
	"kvschool/internal/testutil"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "wordcount":
		if err := runWordCount(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
	case "load":
		if err := runLoad(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
	case "bench":
		if err := runCdrBench(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "kvtool <команда> [аргументы]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Команды:")
	fmt.Fprintln(os.Stderr, "  wordcount  -in <файл> [-store memmap|skiplist|lsm]   выполнить map/reduce wordcount")
	fmt.Fprintln(os.Stderr, "  load       -count <N> [-zipf <S>] [-store ...]   запустить нагрузочное тестирование")
}

func runWordCount(args []string) error {
	fs := flag.NewFlagSet("wordcount", flag.ContinueOnError)
	inPath := fs.String("in", "", "входной текстовый файл")
	storeKind := fs.String("store", "memmap", "тип хранилища: memmap|skiplist|lsm")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inPath == "" {
		return fmt.Errorf("отсутствует параметр -in")
	}
	b, err := os.ReadFile(*inPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения: %w", err)
	}

	ctx := context.Background()
	st, err := initStore(*storeKind, "")
	if err != nil {
		return err
	}
	defer st.Close()

	out, err := mapreduce.Run(ctx, bytes.NewReader(b), st, mapreduce.WordCountMapper, mapreduce.SumVarintReducer)
	if err != nil {
		return err
	}
	defer out.Close()

	it, err := out.Scan(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("ошибка scan: %w", err)
	}
	defer it.Close()

	for {
		p, ok, err := it.Next()
		if err != nil {
			return fmt.Errorf("ошибка итерации: %w", err)
		}
		if !ok {
			break
		}
		x, n := binary.Varint(p.Value)
		if n <= 0 {
			return fmt.Errorf("некорректный varint для ключа=%q", string(p.Key))
		}
		fmt.Printf("%s\t%d\n", string(p.Key), x)
	}
	return nil
}

func runLoad(args []string) error {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	count := fs.Int("count", 10000, "количество операций")
	zipf := fs.Float64("zipf", 0, "параметр s для Zipf (0 для равномерного, >1.0 для перекошенного)")
	storeKind := fs.String("store", "memmap", "тип хранилища: memmap|skiplist|lsm")
	dbDir := fs.String("dir", "", "директория для LSM DB (по умолчанию временная)")
	report := fs.Bool("report", false, "show top talkers report")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, err := initStore(*storeKind, *dbDir)
	if err != nil {
		return err
	}
	defer st.Close()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var keyGen testutil.KeyGenerator
	if *zipf > 1.0 {
		// 1000 items dictionary for zipf
		keyGen = testutil.NewZipfGenerator(rng, *zipf, 1.0, 1000, 16)
		fmt.Printf("Нагрузка: Zipf(s=%.1f) на 1000 элементов\n", *zipf)
	} else {
		keyGen = &testutil.UniformGenerator{Rng: rng, Len: 16}
		fmt.Printf("Нагрузка: Равномерная (Uniform)\n")
	}

	var cms *stream.CountMinSketch
	groundTruth := make(map[string]int)
	if *report {
		cms = stream.NewCountMinSketch(2000, 5)
	}

	start := time.Now()
	ctx := context.Background()

	// Simple Mixed Workload: 50% Put, 50% Get
	for i := 0; i < *count; i++ {
		key := keyGen.Next()
		if err := st.Put(ctx, key, []byte("val")); err != nil {
			return err
		}

		if *report {
			cms.Add(key)
			groundTruth[string(key)]++
		}
	}
	fmt.Printf("Done %d ops in %v\n", *count, time.Since(start))

	if *report {
		printReport(cms, groundTruth)
	}
	return nil
}

func printReport(cms *stream.CountMinSketch, truth map[string]int) {
	fmt.Println("\n=== Top Talkers (CMS vs Real) ===")
	type pair struct {
		k    string
		real int
		est  uint64
	}
	var data []pair
	for k, v := range truth {
		est, _ := cms.Estimate([]byte(k))
		data = append(data, pair{k, v, est})
	}
	sort.Slice(data, func(i, j int) bool { return data[i].real > data[j].real })

	fmt.Printf("%-20s | %-6s | %-6s | %s\n", "Key", "Real", "Est", "Error")
	for i := 0; i < 10 && i < len(data); i++ {
		d := data[i]
		errP := float64(d.est-uint64(d.real)) / float64(d.real) * 100
		fmt.Printf("%-20s | %-6d | %-6d | +%.1f%%\n", d.k, d.real, d.est, errP)
	}
}

func runCdrBench(args []string) error {
	fs := flag.NewFlagSet("cdr-bench", flag.ContinueOnError)
	inPath := fs.String("in", "", "входной CSV файл")
	storeKind := fs.String("store", "lsm", "тип хранилища (lsm рекомендуется)")
	dbDir := fs.String("dir", "", "директория для LSM DB (по умолчанию временная)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *inPath == "" {
		return fmt.Errorf("отсутствует параметр -in")
	}

	st, err := initStore(*storeKind, *dbDir)
	if err != nil {
		return err
	}
	defer st.Close()

	f, err := os.Open(*inPath)
	if err != nil {
		return fmt.Errorf("ошибка открытия CSV: %w", err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	ctx := context.Background()
	count := 0
	start := time.Now()

	if _, err := csvReader.Read(); err != nil {
		return err
	}

	fmt.Printf("Начало загрузки CDR из %s в %s...\n", *inPath, *storeKind)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		key := []byte(record[0] + "_" + record[1])
		value := []byte(strings.Join(record[2:], ","))

		if err := st.Put(ctx, key, value); err != nil {
			return fmt.Errorf("ошибка Put CDR: %w", err)
		}
		count++
	}

	dur := time.Since(start)
	fmt.Printf("Загружено %d записей за %v (%.1f rec/s)\n", count, dur, float64(count)/dur.Seconds())
	return nil
}

func initStore(kind, dir string) (kv.Store, error) {
	switch kind {
	case "memmap":
		return memmap.New(), nil
	case "skiplist":
		return memSkipListDefault(), nil
	case "lsm":
		if dir == "" {
			tmpDir, err := os.MkdirTemp("", "kvtool-lsm-")
			if err != nil {
				return nil, err
			}
			dir = tmpDir
		}
		fmt.Printf("Используется LSM-хранилище в директории: %s\n", dir)
		return lsmstore.Open(lsmstore.Options{Dir: dir})
	default:
		return nil, fmt.Errorf("неизвестное хранилище %q", kind)
	}
}

var _ kv.Store = (*memmap.Store)(nil)
