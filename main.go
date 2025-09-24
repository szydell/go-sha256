package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// Buffer size for reading files - 64KB chunks for memory efficiency
	bufferSize = 64 * 1024
	// Default number of workers for concurrent processing
	defaultWorkers = 4
)

// FileResult represents the result of SHA256 calculation for a single file
type FileResult struct {
	Path     string
	Hash     string
	Size     int64
	Duration time.Duration
	Error    error
}

// FileProcessor handles SHA256 calculation for files
type FileProcessor struct {
	workerCount int
}

// NewFileProcessor creates a new file processor with the specified worker count
func NewFileProcessor(workers int) *FileProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > defaultWorkers {
			workers = defaultWorkers
		}
	}
	return &FileProcessor{workerCount: workers}
}

// calculateSHA256 calculates the SHA256 hash of a single file using a memory-efficient approach
func (fp *FileProcessor) calculateSHA256(filePath string) FileResult {
	startTime := time.Now()

	result := FileResult{
		Path: filePath,
	}

	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to open file: %w", err)
		return result
	}
	defer func() { _ = file.Close() }()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		result.Error = fmt.Errorf("failed to get file stats: %w", err)
		return result
	}
	result.Size = stat.Size()

	// Create SHA256 hasher
	hasher := sha256.New()

	// Use a buffer to read a file in chunks for memory efficiency with large files
	buffer := make([]byte, bufferSize)

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

	// Get the final hash
	hashBytes := hasher.Sum(nil)
	result.Hash = fmt.Sprintf("%x", hashBytes)
	result.Duration = time.Since(startTime)

	return result
}

// ProcessFiles processes multiple files concurrently
func (fp *FileProcessor) ProcessFiles(filePaths []string) []FileResult {
	if len(filePaths) == 0 {
		return []FileResult{}
	}

	// Create channels for work distribution
	jobs := make(chan string, len(filePaths))
	results := make(chan FileResult, len(filePaths))

	// Start workers
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

	// Send jobs
	go func() {
		for _, filePath := range filePaths {
			jobs <- filePath
		}
		close(jobs)
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []FileResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

// readFileList reads a list of files from a text file (one per line)
func readFileList(listPath string) ([]string, error) {
	file, err := os.Open(listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file list: %w", err)
	}
	defer func() { _ = file.Close() }()

	var files []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") { // Skip empty lines and comments
			files = append(files, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file list: %w", err)
	}

	return files, nil
}

// formatSize formats file size in human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// printUsage prints the usage instructions
func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, `Usage: %s [options] <file1> [file2] ...
       %s [options] -list <file_list.txt>

Calculate SHA256 checksums for files, optimized for large files up to 5TiB.

Options:
  -list <file>     Read file paths from a text file (one per line)
  -workers <num>   Number of concurrent workers (default: %d, max: CPU cores)
  -h, -help        Show this help message

Examples:
  %s file1.txt file2.bin
  %s -list files.txt
  %s -workers 8 largefile.iso
  
File list format (files.txt):
  /path/to/file1.txt
  /path/to/file2.bin
  # Comments starting with # are ignored
  /path/to/file3.dat

`, os.Args[0], os.Args[0], defaultWorkers, os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var filePaths []string
	var workers = defaultWorkers

	// Parse command line arguments
	i := 1
	for i < len(os.Args) {
		arg := os.Args[i]

		switch arg {
		case "-h", "-help", "--help":
			printUsage()
			os.Exit(0)
		case "-list":
			if i+1 >= len(os.Args) {
				_, _ = fmt.Fprintf(os.Stderr, "Error: -list requires a file path\n")
				os.Exit(1)
			}
			listPath := os.Args[i+1]
			files, err := readFileList(listPath)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error reading file list: %v\n", err)
				os.Exit(1)
			}
			filePaths = append(filePaths, files...)
			i += 2
		case "-workers":
			if i+1 >= len(os.Args) {
				_, _ = fmt.Fprintf(os.Stderr, "Error: -workers requires a number\n")
				os.Exit(1)
			}
			if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: invalid worker count: %v\n", err)
				os.Exit(1)
			}
			if workers <= 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Error: worker count must be positive\n")
				os.Exit(1)
			}
			i += 2
		default:
			if strings.HasPrefix(arg, "-") {
				_, _ = fmt.Fprintf(os.Stderr, "Error: unknown option %s\n", arg)
				os.Exit(1)
			}
			filePaths = append(filePaths, arg)
			i++
		}
	}

	if len(filePaths) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: no files specified\n")
		printUsage()
		os.Exit(1)
	}

	// Validate that all files exist
	var validFiles []string
	for _, path := range filePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: file does not exist: %s\n", path)
			continue
		}
		validFiles = append(validFiles, path)
	}

	if len(validFiles) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Error: no valid files to process\n")
		os.Exit(1)
	}

	// Create a processor and calculate checksums
	processor := NewFileProcessor(workers)

	fmt.Printf("Processing %d files with %d workers...\n\n", len(validFiles), processor.workerCount)

	startTime := time.Now()
	results := processor.ProcessFiles(validFiles)
	totalTime := time.Since(startTime)

	// Print results
	var totalSize int64
	var successCount int

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("ERROR: %s - %v\n", result.Path, result.Error)
		} else {
			fmt.Printf("%s  %s (%s, %v)\n", result.Hash, filepath.Base(result.Path), formatSize(result.Size), result.Duration)
			totalSize += result.Size
			successCount++
		}
	}

	// Print summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Files processed: %d/%d\n", successCount, len(validFiles))
	fmt.Printf("  Total size: %s\n", formatSize(totalSize))
	fmt.Printf("  Total time: %v\n", totalTime)
	if totalTime > 0 && totalSize > 0 {
		throughput := float64(totalSize) / totalTime.Seconds() / 1024 / 1024
		fmt.Printf("  Throughput: %.2f MB/s\n", throughput)
	}

	// Exit with error code if any files failed
	if successCount < len(validFiles) {
		os.Exit(1)
	}
}
