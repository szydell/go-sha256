package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	errCLIExit      = errors.New("CLI exit")
	errNoFilesGiven = errors.New("no files specified")
)

type cliOptions struct {
	filePaths    []string
	workers      int
	chunkSize    int
	chunkSizeSet bool
	benchmark    bool
}

func runApplication(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	options, err := parseCLIArgs(args)
	if errors.Is(err, errCLIExit) {
		return 0
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if errors.Is(err, errNoFilesGiven) {
			printUsage()
		}
		return 1
	}
	return processCLI(options)
}

func parseCLIArgs(args []string) (cliOptions, error) {
	options := cliOptions{workers: defaultWorkers, chunkSize: defaultChunkSize}
	for index := 0; index < len(args); {
		switch args[index] {
		case "-v", "-version", "--version":
			fmt.Println(version)
			return options, errCLIExit
		case "-h", "-help", "--help":
			printUsage()
			return options, errCLIExit
		}

		next, err := parseCLIArgument(args, index, &options)
		if err != nil {
			return options, err
		}
		index = next
	}

	if options.benchmark && options.chunkSizeSet {
		return options, fmt.Errorf("-benchmark cannot be combined with -chunk-size")
	}
	if len(options.filePaths) == 0 {
		return options, errNoFilesGiven
	}
	return options, nil
}

func parseCLIArgument(args []string, index int, options *cliOptions) (int, error) {
	switch args[index] {
	case "-benchmark":
		options.benchmark = true
		return index + 1, nil
	case "-chunk-size":
		value, next, err := requiredOptionValue(args, index, "-chunk-size requires a value")
		if err != nil {
			return index, err
		}
		options.chunkSize, err = parseChunkSize(value)
		if err != nil {
			return index, fmt.Errorf("invalid chunk size: %w", err)
		}
		options.chunkSizeSet = true
		return next, nil
	case "-list":
		value, next, err := requiredOptionValue(args, index, "-list requires a file path")
		if err != nil {
			return index, err
		}
		files, err := readFileList(value)
		if err != nil {
			return index, fmt.Errorf("reading file list: %w", err)
		}
		options.filePaths = append(options.filePaths, files...)
		return next, nil
	case "-workers":
		value, next, err := requiredOptionValue(args, index, "-workers requires a number")
		if err != nil {
			return index, err
		}
		options.workers, err = strconv.Atoi(value)
		if err != nil {
			return index, fmt.Errorf("invalid worker count: %w", err)
		}
		if options.workers <= 0 {
			return index, fmt.Errorf("worker count must be positive")
		}
		return next, nil
	default:
		if strings.HasPrefix(args[index], "-") {
			return index, fmt.Errorf("unknown option %s", args[index])
		}
		options.filePaths = append(options.filePaths, args[index])
		return index + 1, nil
	}
}

func requiredOptionValue(args []string, index int, message string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s", message)
	}
	return args[index+1], index + 2, nil
}

func processCLI(options cliOptions) int {
	validFiles := existingFiles(options.filePaths)
	if len(validFiles) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid files to process")
		return 1
	}

	var results []FileResult
	var totalTime time.Duration
	if options.benchmark {
		var err error
		results, totalTime, err = benchmarkCLI(validFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Benchmark failed: %v\n", err)
			return 1
		}
	} else {
		results, totalTime = processCLINormally(validFiles, options)
	}

	return printResults(results, len(validFiles), totalTime, options.benchmark)
}

func existingFiles(filePaths []string) []string {
	validFiles := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: file does not exist: %s\n", path)
			continue
		}
		validFiles = append(validFiles, path)
	}
	return validFiles
}

func benchmarkCLI(filePaths []string) ([]FileResult, time.Duration, error) {
	fmt.Printf("Benchmarking %d files with full reads for each chunk size.\n", len(filePaths))
	fmt.Println("Calibration is sequential; -workers does not affect benchmark.")
	startTime := time.Now()
	report, err := benchmarkFiles(filePaths)
	duration := time.Since(startTime)
	if err != nil {
		return nil, duration, err
	}
	printBenchmarkReport(report)
	return report.Results, duration, nil
}

func processCLINormally(filePaths []string, options cliOptions) ([]FileResult, time.Duration) {
	processor := newFileProcessor(options.workers, options.chunkSize)
	fmt.Printf("Processing %d files with %d workers...\n\n", len(filePaths), processor.workerCount)
	startTime := time.Now()
	results := processor.ProcessFiles(filePaths)
	return results, time.Since(startTime)
}

func printResults(results []FileResult, expectedCount int, totalTime time.Duration, benchmark bool) int {
	var totalSize int64
	var successCount int
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("ERROR: %s - %v\n", result.Path, result.Error)
			continue
		}
		fmt.Printf("%s  %s (%s, %v)\n", result.Hash, filepath.Base(result.Path), formatSize(result.Size), result.Duration)
		totalSize += result.Size
		successCount++
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Files processed: %d/%d\n", successCount, expectedCount)
	fmt.Printf("  Total size: %s\n", formatSize(totalSize))
	fmt.Printf("  Total time: %v\n", totalTime)
	if !benchmark && totalTime > 0 && totalSize > 0 {
		throughput := float64(totalSize) / totalTime.Seconds() / 1024 / 1024
		fmt.Printf("  Throughput: %.2f MB/s\n", throughput)
	}
	if successCount < expectedCount {
		return 1
	}
	return 0
}
