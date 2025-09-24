package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestCalculateSHA256(t *testing.T) {
	processor := NewFileProcessor(2)
	
	// Create a temporary test file
	tempFile := "/tmp/test_sha256.txt"
	content := "Hello, World!"
	
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)
	
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

func TestProcessMultipleFiles(t *testing.T) {
	processor := NewFileProcessor(2)
	
	// Create multiple test files
	testFiles := []string{"/tmp/test1.txt", "/tmp/test2.txt", "/tmp/test3.txt"}
	testContents := []string{"File 1", "File 2 content", "File 3 has different content"}
	
	for i, file := range testFiles {
		err := os.WriteFile(file, []byte(testContents[i]), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		defer os.Remove(file)
	}
	
	// Process files
	results := processor.ProcessFiles(testFiles)
	
	if len(results) != len(testFiles) {
		t.Errorf("Expected %d results, got %d", len(testFiles), len(results))
	}
	
	// Check that all files were processed without error
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Error processing file %s: %v", result.Path, result.Error)
		}
		if result.Hash == "" {
			t.Errorf("Empty hash for file %s", result.Path)
		}
	}
}

func TestReadFileList(t *testing.T) {
	// Create a test file list
	listFile := "/tmp/filelist.txt"
	listContent := `# This is a comment
/tmp/test1.txt
/tmp/test2.txt

# Another comment
/tmp/test3.txt`
	
	err := os.WriteFile(listFile, []byte(listContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create file list: %v", err)
	}
	defer os.Remove(listFile)
	
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
	// Create a larger test file (1MB)
	tempFile := "/tmp/large_test.bin"
	size := 1024 * 1024 // 1MB
	
	file, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}
	
	// Write predictable data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	for i := 0; i < size/len(data); i++ {
		_, err := file.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to test file: %v", err)
		}
	}
	file.Close()
	defer os.Remove(tempFile)
	
	processor := NewFileProcessor(2)
	result := processor.calculateSHA256(tempFile)
	
	if result.Error != nil {
		t.Fatalf("Error calculating SHA256 for large file: %v", result.Error)
	}
	
	if result.Size != int64(size) {
		t.Errorf("Size mismatch. Expected: %d, Got: %d", size, result.Size)
	}
	
	if len(result.Hash) != 64 { // SHA256 hash is 64 hex characters
		t.Errorf("Invalid hash length. Expected: 64, Got: %d", len(result.Hash))
	}
	
	// Verify the hash is hexadecimal
	for _, c := range result.Hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Hash contains non-hexadecimal character: %c", c)
			break
		}
	}
}

// Benchmark test for performance
func BenchmarkSHA256Calculation(b *testing.B) {
	// Create a test file
	tempFile := "/tmp/benchmark_test.bin"
	size := 64 * 1024 // 64KB
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	
	err := os.WriteFile(tempFile, data, 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark test file: %v", err)
	}
	defer os.Remove(tempFile)
	
	processor := NewFileProcessor(1)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := processor.calculateSHA256(tempFile)
		if result.Error != nil {
			b.Fatalf("Error in benchmark: %v", result.Error)
		}
	}
}