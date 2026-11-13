package main

import "os"

const (
	defaultChunkSize = 64 * 1024
	minChunkSize     = 1 * 1024
	maxChunkSize     = 16 * 1024 * 1024
	defaultWorkers   = 4
)

var version = "dev"

func main() {
	os.Exit(runApplication(os.Args[1:]))
}
