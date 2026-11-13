package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// FileResult contains the SHA-256 result and processing details for one file.
type FileResult struct {
	Path     string
	Hash     string
	Size     int64
	Duration time.Duration
	Error    error
}

// FileProcessor calculates SHA-256 checksums for files.
type FileProcessor struct {
	workerCount int
	chunkSize   int
}

// NewFileProcessor creates a processor using the default read chunk size.
func NewFileProcessor(workers int) *FileProcessor {
	return newFileProcessor(workers, defaultChunkSize)
}

func newFileProcessor(workers, chunkSize int) *FileProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > defaultWorkers {
			workers = defaultWorkers
		}
	}
	return &FileProcessor{workerCount: workers, chunkSize: chunkSize}
}

func (fp *FileProcessor) calculateSHA256(filePath string) FileResult {
	return calculateSHA256WithChunkSize(filePath, fp.chunkSize)
}

func calculateSHA256WithChunkSize(filePath string, chunkSize int) FileResult {
	startTime := time.Now()
	result := FileResult{Path: filePath}
	if chunkSize <= 0 {
		result.Error = fmt.Errorf("chunk size must be positive")
		return result
	}

	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		return result
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		result.Error = fmt.Errorf("failed to get file stats: %w", err)
		return result
	}
	result.Size = stat.Size()

	hasher := sha256.New()
	buffer := make([]byte, chunkSize)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hasher.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Error = fmt.Errorf("failed to read file: %w", err)
			return result
		}
	}

	result.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	result.Duration = time.Since(startTime)
	return result
}

// ProcessFiles calculates checksums concurrently for the supplied paths.
func (fp *FileProcessor) ProcessFiles(filePaths []string) []FileResult {
	if len(filePaths) == 0 {
		return []FileResult{}
	}

	jobs := make(chan string, len(filePaths))
	results := make(chan FileResult, len(filePaths))
	var wg sync.WaitGroup
	for i := 0; i < fp.workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				results <- fp.calculateSHA256(filePath)
			}
		}()
	}

	go func() {
		for _, filePath := range filePaths {
			jobs <- filePath
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []FileResult
	for result := range results {
		allResults = append(allResults, result)
	}
	return allResults
}
