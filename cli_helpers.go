package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
		if line != "" && !strings.HasPrefix(line, "#") {
			files = append(files, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file list: %w", err)
	}
	return files, nil
}

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

func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, `Usage: %s [options] <file1> [file2] ...
       %s [options] -list <file_list.txt>

Calculate SHA256 checksums for files, optimized for large files up to 5TiB.
Project: https://github.com/szydell/go-sha256

Options:
  -list <file>     Read file paths from a text file (one per line)
  -workers <num>   Number of concurrent workers (default: %d)
  -chunk-size <n>  Read buffer size (e.g. 64KiB, 1MiB; default: 64KiB)
  -benchmark       Compare full-file reads using 64KiB, 256KiB, 1MiB, 4MiB, 16MiB
                   Reads every file %d times and can cause significant I/O
  -v, -version     Show version and exit
  -h, -help        Show this help message

Examples:
  %s file1.txt file2.bin
  %s -list files.txt
  %s -workers 8 largefile.iso
  %s -chunk-size 1MiB largefile.iso
  %s -benchmark largefile.iso

File list format (files.txt):
  /path/to/file1.txt
  /path/to/file2.bin
  # Comments starting with # are ignored
  /path/to/file3.dat

`, os.Args[0], os.Args[0], defaultWorkers, len(benchmarkChunkSizes), os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
