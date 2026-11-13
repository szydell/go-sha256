---
date: 2026-11-13
heist: sha256-chunk-calibration
phase: the-hit
tags: [sha256, performance, benchmarking]
---

# Calibrate streaming reads on target files

## Discovery

The target workload consists of a small number of very large files, so a small in-memory benchmark would not represent storage behavior. A useful comparison must measure actual file reads while making the extra I/O cost explicit.

## Solution

Benchmark representative files on the target platform with full sequential reads at several chunk sizes. Calculate and compare the checksum on every pass, select a recommendation from the aggregate throughput, and reuse the checksum from that selected pass instead of reading the files again. Report that later passes may benefit from the operating system cache.

## Project Commandment

When tuning streaming-file throughput, benchmark representative target files and disclose repeated-read and cache effects; reuse the checksum from the recommended pass to avoid an unnecessary extra read.
