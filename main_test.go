package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculateSHA256(t *testing.T) {
	processor := NewFileProcessor(2)

	tempFile := filepath.Join(t.TempDir(), "input.txt")
	content := "Hello, World!"

	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected hash
	hasher := sha256.New()
	hasher.Write([]byte(content))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Test our implementation
	result := processor.calculateSHA256(tempFile)

	if result.Error != nil {
		t.Fatalf("Error calculating SHA256: %v", result.Error)
	}

	if result.Hash != expectedHash {
		t.Errorf("Hash mismatch. Expected: %s, Got: %s", expectedHash, result.Hash)
	}

	if result.Size != int64(len(content)) {
		t.Errorf("Size mismatch. Expected: %d, Got: %d", len(content), result.Size)
	}

	if result.Path != tempFile {
		t.Errorf("Path mismatch. Expected: %s, Got: %s", tempFile, result.Path)
	}

	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestCalculateSHA256EmptyFile(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(tempFile, nil, 0600); err != nil {
		t.Fatal(err)
	}

	result := NewFileProcessor(1).calculateSHA256(tempFile)
	if result.Error != nil {
		t.Fatalf("calculateSHA256 returned error: %v", result.Error)
	}
	if result.Hash != fmt.Sprintf("%x", sha256.Sum256(nil)) {
		t.Errorf("unexpected hash for empty file: %s", result.Hash)
	}
	if result.Size != 0 {
		t.Errorf("expected empty file size 0, got %d", result.Size)
	}
}

func TestCalculateSHA256MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")

	result := NewFileProcessor(1).calculateSHA256(path)
	if result.Error == nil {
		t.Fatal("expected an error for a missing file")
	}
	if result.Path != path {
		t.Errorf("expected result path %q, got %q", path, result.Path)
	}
}

func TestProcessMultipleFiles(t *testing.T) {
	processor := NewFileProcessor(2)

	tempDir := t.TempDir()
	testFiles := []string{
		filepath.Join(tempDir, "test1.txt"),
		filepath.Join(tempDir, "test2.txt"),
		filepath.Join(tempDir, "test3.txt"),
	}
	testContents := []string{"File 1", "File 2 content", "File 3 has different content"}

	for i, file := range testFiles {
		err := os.WriteFile(file, []byte(testContents[i]), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Process files
	results := processor.ProcessFiles(testFiles)

	if len(results) != len(testFiles) {
		t.Errorf("Expected %d results, got %d", len(testFiles), len(results))
	}

	expectedHashes := make(map[string]string, len(testFiles))
	for i, file := range testFiles {
		expectedHashes[file] = fmt.Sprintf("%x", sha256.Sum256([]byte(testContents[i])))
	}
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Error processing file %s: %v", result.Path, result.Error)
			continue
		}
		if result.Hash != expectedHashes[result.Path] {
			t.Errorf("Unexpected hash for %s: got %s, want %s", result.Path, result.Hash, expectedHashes[result.Path])
		}
		delete(expectedHashes, result.Path)
	}
	if len(expectedHashes) != 0 {
		t.Errorf("missing results for files: %v", expectedHashes)
	}
}

func TestProcessFilesEmptyInput(t *testing.T) {
	results := NewFileProcessor(1).ProcessFiles(nil)
	if len(results) != 0 {
		t.Errorf("expected no results for empty input, got %d", len(results))
	}
}

func TestReadFileList(t *testing.T) {
	listFile := filepath.Join(t.TempDir(), "filelist.txt")
	listContent := `# This is a comment
/tmp/test1.txt
/tmp/test2.txt

# Another comment
/tmp/test3.txt`

	err := os.WriteFile(listFile, []byte(listContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create file list: %v", err)
	}

	files, err := readFileList(listFile)
	if err != nil {
		t.Fatalf("Error reading file list: %v", err)
	}

	expected := []string{"/tmp/test1.txt", "/tmp/test2.txt", "/tmp/test3.txt"}
	if len(files) != len(expected) {
		t.Errorf("Expected %d files, got %d", len(expected), len(files))
	}

	for i, file := range files {
		if file != expected[i] {
			t.Errorf("Expected file %s at position %d, got %s", expected[i], i, file)
		}
	}
}

func TestReadFileListRejectsOverlongLine(t *testing.T) {
	listFile := filepath.Join(t.TempDir(), "filelist.txt")
	if err := os.WriteFile(listFile, []byte(strings.Repeat("a", 70*1024)), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := readFileList(listFile); err == nil {
		t.Fatal("expected an error for a line exceeding the scanner limit")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{5497558138880, "5.0 TB"}, // 5 TiB
	}

	for _, test := range tests {
		result := formatSize(test.bytes)
		if result != test.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestNewFileProcessor(t *testing.T) {
	// Test default worker count
	processor := NewFileProcessor(0)
	if processor.workerCount <= 0 {
		t.Error("Worker count should be positive")
	}

	// Test specific worker count
	processor = NewFileProcessor(8)
	if processor.workerCount != 8 {
		t.Errorf("Expected 8 workers, got %d", processor.workerCount)
	}
}

func TestLargeFileProcessing(t *testing.T) {
	size := 1024 * 1024 // 1MB
	tempFile := filepath.Join(t.TempDir(), "large_test.bin")
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	processor := NewFileProcessor(2)
	result := processor.calculateSHA256(tempFile)

	if result.Error != nil {
		t.Fatalf("Error calculating SHA256 for large file: %v", result.Error)
	}

	if result.Size != int64(size) {
		t.Errorf("Size mismatch. Expected: %d, Got: %d", size, result.Size)
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(data))
	if result.Hash != expectedHash {
		t.Errorf("Hash mismatch. Expected: %s, Got: %s", expectedHash, result.Hash)
	}
}

// Benchmark test for performance
func BenchmarkSHA256Calculation(b *testing.B) {
	// Create a test file
	tempFile := filepath.Join(b.TempDir(), "benchmark_test.bin")
	size := 64 * 1024 // 64KB
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := os.WriteFile(tempFile, data, 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark test file: %v", err)
	}

	processor := NewFileProcessor(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := processor.calculateSHA256(tempFile)
		if result.Error != nil {
			b.Fatalf("Error in benchmark: %v", result.Error)
		}
	}
}
