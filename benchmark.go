package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var benchmarkChunkSizes = [...]int{
	64 * 1024,
	256 * 1024,
	1 * 1024 * 1024,
	4 * 1024 * 1024,
	16 * 1024 * 1024,
}

type benchmarkMeasurement struct {
	ChunkSize int
	Duration  time.Duration
	MBPerSec  float64
}

type benchmarkFileReport struct {
	Path         string
	Size         int64
	Measurements []benchmarkMeasurement
}

type benchmarkReport struct {
	Files                []benchmarkFileReport
	Results              []FileResult
	RecommendedChunkSize int
	MBPerSec             float64
}

func parseChunkSize(value string) (int, error) {
	multiplier := int64(1)
	number := value
	switch {
	case strings.HasSuffix(value, "KiB"):
		multiplier = 1024
		number = strings.TrimSuffix(value, "KiB")
	case strings.HasSuffix(value, "MiB"):
		multiplier = 1024 * 1024
		number = strings.TrimSuffix(value, "MiB")
	}

	parsed, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("expected a byte count with optional KiB or MiB suffix")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("must be at least %s", formatChunkSize(minChunkSize))
	}
	if parsed > int64(maxChunkSize)/multiplier {
		return 0, fmt.Errorf("must not exceed %s", formatChunkSize(maxChunkSize))
	}
	size := parsed * multiplier
	if size < int64(minChunkSize) {
		return 0, fmt.Errorf("must be at least %s", formatChunkSize(minChunkSize))
	}
	return int(size), nil
}

func benchmarkFiles(filePaths []string) (benchmarkReport, error) {
	for _, path := range filePaths {
		info, err := os.Stat(path)
		if err != nil {
			return benchmarkReport{}, fmt.Errorf("cannot stat %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return benchmarkReport{}, fmt.Errorf("%q is not a regular file", path)
		}
	}

	report := benchmarkReport{
		Files:   make([]benchmarkFileReport, len(filePaths)),
		Results: make([]FileResult, len(filePaths)),
	}
	resultsByChunk := make([][]FileResult, len(benchmarkChunkSizes))
	durationsByChunk := make([]time.Duration, len(benchmarkChunkSizes))
	var totalSize int64

	for candidateIndex, chunkSize := range benchmarkChunkSizes {
		resultsByChunk[candidateIndex] = make([]FileResult, len(filePaths))
		for fileIndex, path := range filePaths {
			result := calculateSHA256WithChunkSize(path, chunkSize)
			if result.Error != nil {
				return report, fmt.Errorf("%q with %s: %w", path, formatChunkSize(chunkSize), result.Error)
			}
			if candidateIndex > 0 {
				if err := validateBenchmarkResult(resultsByChunk[0][fileIndex], result); err != nil {
					return report, err
				}
			} else {
				report.Files[fileIndex].Path = path
				report.Files[fileIndex].Size = result.Size
				totalSize += result.Size
			}

			resultsByChunk[candidateIndex][fileIndex] = result
			durationsByChunk[candidateIndex] += result.Duration
			measurement := benchmarkMeasurement{
				ChunkSize: chunkSize,
				Duration:  result.Duration,
				MBPerSec:  throughputMBPerSec(result.Size, result.Duration),
			}
			report.Files[fileIndex].Measurements = append(report.Files[fileIndex].Measurements, measurement)
		}
	}

	bestIndex := bestBenchmarkIndex(totalSize, durationsByChunk)

	report.RecommendedChunkSize = benchmarkChunkSizes[bestIndex]
	report.MBPerSec = throughputMBPerSec(totalSize, durationsByChunk[bestIndex])
	copy(report.Results, resultsByChunk[bestIndex])
	return report, nil
}

func bestBenchmarkIndex(totalSize int64, durations []time.Duration) int {
	bestIndex := 0
	for candidateIndex := 1; candidateIndex < len(durations); candidateIndex++ {
		candidateThroughput := throughputMBPerSec(totalSize, durations[candidateIndex])
		bestThroughput := throughputMBPerSec(totalSize, durations[bestIndex])
		if candidateThroughput > bestThroughput {
			bestIndex = candidateIndex
		}
	}
	return bestIndex
}

func validateBenchmarkResult(previous, current FileResult) error {
	if previous.Size != current.Size {
		return fmt.Errorf("%q changed size during benchmark", current.Path)
	}
	if previous.Hash != current.Hash {
		return fmt.Errorf("%q produced different SHA-256 hashes during benchmark", current.Path)
	}
	return nil
}

func throughputMBPerSec(size int64, duration time.Duration) float64 {
	if size <= 0 || duration <= 0 {
		return 0
	}
	return float64(size) / duration.Seconds() / 1024 / 1024
}

func formatChunkSize(size int) string {
	switch {
	case size%(1024*1024) == 0:
		return fmt.Sprintf("%d MiB", size/(1024*1024))
	case size%1024 == 0:
		return fmt.Sprintf("%d KiB", size/1024)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func printBenchmarkReport(report benchmarkReport) {
	for _, file := range report.Files {
		fmt.Printf("\n%s (%s):\n", file.Path, formatSize(file.Size))
		for _, measurement := range file.Measurements {
			fmt.Printf("  %s: %.2f MB/s (%v)\n", formatChunkSize(measurement.ChunkSize), measurement.MBPerSec, measurement.Duration)
		}
	}
	fmt.Printf("\nRecommended chunk size: %s (combined throughput: %.2f MB/s)\n",
		formatChunkSize(report.RecommendedChunkSize), report.MBPerSec)
}
