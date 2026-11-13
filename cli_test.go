package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIRejectsMalformedWorkerCount(t *testing.T) {
	if output, err := runCLI(t, "-workers", "2x", "unused.txt"); err == nil {
		t.Fatalf("expected malformed worker count to be rejected, output: %s", output)
	}
}

func TestCLIChunkSize(t *testing.T) {
	testFile := writeTestFile(t, "input.txt", []byte("content"))

	output, err := runCLI(t, "-chunk-size", "1MiB", testFile)
	if err != nil {
		t.Fatalf("expected valid chunk size to be accepted: %v\n%s", err, output)
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte("content")))
	if !strings.Contains(string(output), expectedHash) {
		t.Fatalf("expected output to contain hash %s, got:\n%s", expectedHash, output)
	}
}

func TestCLIRejectsInvalidChunkSizes(t *testing.T) {
	testFile := writeTestFile(t, "input.txt", []byte("content"))
	for _, chunkSize := range []string{"0", "-1", "not-a-size", "17MiB", "-18014398509481983KiB", "-17592186044415MiB", "-8796093022209MiB"} {
		t.Run(chunkSize, func(t *testing.T) {
			output, err := runCLI(t, "-chunk-size", chunkSize, testFile)
			if err == nil {
				t.Fatalf("expected invalid chunk size to be rejected, output: %s", output)
			}
			if !strings.Contains(string(output), "invalid chunk size") {
				t.Fatalf("expected chunk size validation error, got:\n%s", output)
			}
		})
	}
}

func TestCLIBenchmarkHashesMultipleFiles(t *testing.T) {
	first := writeTestFile(t, "first.bin", []byte("first file contents"))
	second := writeTestFile(t, "second.bin", []byte("second file contents"))

	output, err := runCLI(t, "-benchmark", first, second)
	if err != nil {
		t.Fatalf("benchmark failed: %v\n%s", err, output)
	}

	for _, content := range [][]byte{[]byte("first file contents"), []byte("second file contents")} {
		expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))
		if !strings.Contains(string(output), expectedHash) {
			t.Errorf("expected benchmark output to contain hash %s, got:\n%s", expectedHash, output)
		}
	}
	if !strings.Contains(string(output), "Recommended chunk size:") {
		t.Fatalf("expected benchmark recommendation, got:\n%s", output)
	}
	for _, path := range []string{first, second} {
		if !strings.Contains(string(output), path) {
			t.Errorf("expected per-file benchmark results for %s, got:\n%s", path, output)
		}
	}
	for _, chunkSize := range []string{"64 KiB:", "16 MiB:"} {
		if !strings.Contains(string(output), chunkSize) {
			t.Errorf("expected benchmark result for %s, got:\n%s", chunkSize, output)
		}
	}
	if strings.Contains(string(output), "\n  Throughput:") {
		t.Errorf("benchmark summary should not report one-pass throughput for a multi-pass run:\n%s", output)
	}
}

func TestBenchmarkRejectsNonRegularInputBeforeReading(t *testing.T) {
	first := writeTestFile(t, "first.bin", []byte("first"))
	directory := t.TempDir()
	report, err := benchmarkFiles([]string{first, directory})
	if err == nil || !strings.Contains(err.Error(), directory) || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected non-regular input error for %s, got %v", directory, err)
	}
	if len(report.Files) > 0 && len(report.Files[0].Measurements) > 0 {
		t.Fatal("benchmark read a file before validating all inputs")
	}
}

func TestCLIRejectsBenchmarkWithChunkSize(t *testing.T) {
	testFile := writeTestFile(t, "input.txt", []byte("content"))
	output, err := runCLI(t, "-benchmark", "-chunk-size", "1MiB", testFile)
	if err == nil {
		t.Fatalf("expected benchmark and chunk size conflict to be rejected, output: %s", output)
	}
}

func TestValidateBenchmarkResultRejectsDifferentHashes(t *testing.T) {
	first := FileResult{Path: "file.bin", Size: 10, Hash: "first"}
	second := FileResult{Path: "file.bin", Size: 10, Hash: "second"}
	if err := validateBenchmarkResult(first, second); err == nil {
		t.Fatal("expected different hashes to be rejected")
	}
}

func TestValidateBenchmarkResultRejectsDifferentSizes(t *testing.T) {
	first := FileResult{Path: "file.bin", Size: 10, Hash: "same"}
	second := FileResult{Path: "file.bin", Size: 11, Hash: "same"}
	if err := validateBenchmarkResult(first, second); err == nil {
		t.Fatal("expected different sizes to be rejected")
	}
}

func TestBestBenchmarkIndexUsesAggregateElapsedTime(t *testing.T) {
	durations := []time.Duration{3 * time.Second, time.Second, 2 * time.Second}
	if got := bestBenchmarkIndex(300, durations); got != 1 {
		t.Fatalf("bestBenchmarkIndex() = %d, want 1", got)
	}
}

func runCLI(t *testing.T, args ...string) ([]byte, error) {
	return runCLIContext(t, context.Background(), args...)
}

func runCLIContext(t *testing.T, ctx context.Context, args ...string) ([]byte, error) {
	t.Helper()
	encodedArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIHelper$")
	cmd.Env = append(os.Environ(),
		"GO_SHA256_CLI_HELPER=1",
		"GO_SHA256_CLI_ARGS="+string(encodedArgs),
	)
	return cmd.CombinedOutput()
}

func writeTestFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLIHelper(t *testing.T) {
	if os.Getenv("GO_SHA256_CLI_HELPER") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("GO_SHA256_CLI_ARGS")), &args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Args = append([]string{"go-sha256"}, args...)
	main()
}
