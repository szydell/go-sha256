# go-sha256

A high-performance SHA256 checksum calculator written in Go, optimized for processing huge files up to 5TiB. The program uses only Go standard libraries and leverages multithreading for concurrent file processing.

## Features

- **Large File Support**: Efficiently processes files up to 5TiB using memory-efficient streaming
- **Multithreading**: Concurrent processing of multiple files with configurable worker count
- **Standard Library Only**: Uses only Go standard libraries, no external C dependencies
- **Batch Processing**: Process multiple files from a file list (newline-separated)
- **Memory Efficient**: Uses 64KB buffer size to minimize memory usage while maintaining performance
- **Progress Tracking**: Shows processing time, file sizes, and throughput statistics

## Installation

```bash
git clone https://github.com/szydell/go-sha256
cd go-sha256
go build -o sha256sum .
```

## Usage

### Basic Usage

Calculate SHA256 for single file:
```bash
./sha256sum file.txt
```

Calculate SHA256 for multiple files:
```bash
./sha256sum file1.txt file2.bin file3.dat
```

### Batch Processing

Process files listed in a text file:
```bash
./sha256sum -list files.txt
```

File list format (files.txt):
```
/path/to/file1.txt
/path/to/file2.bin
# Comments starting with # are ignored
/path/to/large-file.iso
```

### Advanced Options

Use specific number of workers:
```bash
./sha256sum -workers 8 large-file.iso
```

Process file list with custom worker count:
```bash
./sha256sum -workers 4 -list files.txt
```

### Command Line Options

- `-list <file>`: Read file paths from a text file (one per line)
- `-workers <num>`: Number of concurrent workers (default: 4, max: CPU cores)
- `-h`, `-help`: Show help message

## Output Format

The program outputs results in the format:
```
<sha256_hash>  <filename> (<file_size>, <processing_time>)
```

Example:
```
c98c24b677eff44860afea6f493bbaec5bb1c4cbb209c6fc2bbb47f66ff2ad31  test.txt (14 B, 54.723µs)
```

## Performance

- **Memory Usage**: Fixed 64KB buffer per worker, minimal memory footprint
- **Throughput**: Optimized for high-throughput processing (typically >1GB/s depending on storage)
- **Scalability**: Configurable worker count for optimal CPU utilization

## Examples

Process a single large file:
```bash
./sha256sum /path/to/large-file.iso
```

Process multiple files concurrently:
```bash
./sha256sum -workers 8 file1.bin file2.bin file3.bin
```

Batch process from file list:
```bash
echo "/path/to/file1.txt" > files.txt
echo "/path/to/file2.bin" >> files.txt
./sha256sum -list files.txt
```

## Testing

Run the test suite:
```bash
go test -v
```

Run benchmarks:
```bash
go test -bench=.
```

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.