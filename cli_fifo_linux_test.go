package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCLIBenchmarkRejectsFIFO(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "input.fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := runCLIContext(t, ctx, "-benchmark", fifo)
	if ctx.Err() != nil {
		t.Fatalf("benchmark blocked on FIFO: %v\n%s", ctx.Err(), output)
	}
	if err == nil || !strings.Contains(string(output), fifo) || !strings.Contains(string(output), "regular file") {
		t.Fatalf("expected non-regular input error, got %v:\n%s", err, output)
	}
}

func TestCLINormalHashingAcceptsFIFO(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "input.fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	writer := exec.CommandContext(ctx, "sh", "-c", `printf 'stream content' > "$1"`, "sh", fifo)
	if err := writer.Start(); err != nil {
		t.Fatal(err)
	}
	output, err := runCLIContext(t, ctx, fifo)
	if err != nil {
		cancel()
	}
	writerErr := writer.Wait()
	if err != nil || writerErr != nil {
		t.Fatalf("hashing FIFO: cli=%v writer=%v context=%v\n%s", err, writerErr, ctx.Err(), output)
	}
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte("stream content")))
	if !strings.Contains(string(output), expected) {
		t.Fatalf("expected stream hash %s, got:\n%s", expected, output)
	}
}

func TestCLIBenchmarkFollowsRegularFileSymlink(t *testing.T) {
	file := writeTestFile(t, "input.bin", []byte("content"))
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if output, err := runCLI(t, "-benchmark", link); err != nil {
		t.Fatalf("expected regular file symlink to be accepted: %v\n%s", err, output)
	}
}
