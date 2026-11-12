package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sourcegraph/conc/iter"
	"github.com/sourcegraph/conc/pool"
	"github.com/sourcegraph/conc/stream"
)

func main() {
	start := time.Now()
	files, err := os.ReadDir("./works")
	if err != nil {
		log.Fatalf("Error reading dir: %v", err)
	}
	maxGoRoutines := 3

	p := pool.NewWithResults[workWordCounts]().WithMaxGoroutines(maxGoRoutines).WithErrors()
	for _, dirEntry := range files {
		p.Go(func() (workWordCounts, error) {
			workStart := time.Now()
			f, err := os.Open(filepath.Join("./works", dirEntry.Name()))
			if err != nil {
				return nil, fmt.Errorf("error opening file: %w", err)
			}
			wc, err := countWords(f)
			slog.Info("Done counting words", "work", dirEntry.Name(), "duration", time.Since(workStart))
			return wc, err
		})
	}
	allWordCounts, err := p.Wait()
	if err != nil {
		slog.Error("error generating word counts", "err", err)
	}

	totalWordCounts := make(workWordCounts)
	s := stream.New().WithMaxGoroutines(maxGoRoutines)
	for _, wwc := range allWordCounts {
		s.Go(func() stream.Callback {
			comWord, comCount := wwc.mostCommonWord()
			slog.Info("unique", "words", len(wwc), "mostCommonWord", comWord, "mostCommonCount", comCount)
			return func() {
				totalWordCounts.add(wwc)
			}
		})
	}
	s.Wait()
	totalCommonWord, totalCommonCount := totalWordCounts.mostCommonWord()
	slog.Info("Done with all word counts",
		"in", time.Since(start),
		"totalUniqueWords", len(totalWordCounts),
		"mostCommonWord", totalCommonWord,
		"mostCommonCount", totalCommonCount)
}

func countWords(in io.ReadCloser) (workWordCounts, error) {
	defer in.Close()
	inputBytes, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	splits := bytes.Fields(inputBytes)
	result := make(workWordCounts)
	mu := &sync.Mutex{}
	iter.ForEach(splits, func(i *[]byte) {
		mu.Lock()
		defer mu.Unlock()
		result[string(*i)]++
	})

	return result, nil
}

type workWordCounts map[string]int

func (wwc workWordCounts) mostCommonWord() (string, int) {
	word := ""
	count := -1

	for candidateWord, candiateCount := range wwc {
		if candiateCount > count {
			word = candidateWord
			count = candiateCount
		}
	}
	return word, count
}

func (wwc workWordCounts) add(other workWordCounts) {
	for word, count := range other {
		wwc[word] = wwc[word] + count
	}
}
