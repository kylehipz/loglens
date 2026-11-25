# loglens: Software requirements specification

Version 4.0 · 22 November 2026

> This file is the source of truth for loglens's requirements. It began as a transcription of the
> SRS v4.0 PDF; [`loglens-srs.pdf`](../loglens-srs.pdf) is now rendered from this file. Edit
> this file, not the PDF.

> **Project note: build order.** loglens is built naive-first. The first
> version only has to produce correct output: whole-file reads, brute-force statistics and no
> concurrency are fine. FR-2's streaming and 1 MiB line-limit rules, and all of section 5, are met
> afterwards, one at a time. The output (counts, percentiles, orderings, messages, exit codes) must
> already match this spec in the first version. The finished tool must meet every requirement
> below. The efficiency requirements in sections 5.1–5.3 (NFR-1 to NFR-5, NFR-8 and NFR-9) are
> the project's high-value learning goals; the naive version's measurements are the baseline
> they're compared against.

> **Changes from the original v4.0.** Follow mode was removed to reduce scope (25 November 2026):
> FR-18 to FR-23, NFR-6, NFR-7, NFR-10, data set D4, and the `--follow`, `--interval` and
> `--log-level` flags. Removed IDs are kept as gaps and not reused.

**Product.** A command-line tool, written in Go, that analyzes access logs and reports traffic,
errors and latency over one or more log files.

**This document.** What the tool must do (functional requirements, FR) and how well it must do it
(non-functional requirements, NFR), each with acceptance criteria. Design and implementation are
deliberately left open.

**Contents.** [1 Introduction](#1-introduction) ·
[2 Input format](#2-input-format) ·
[3 Functional requirements](#3-functional-requirements) ·
[4 Output formats](#4-output-formats) ·
[5 Non-functional requirements](#5-non-functional-requirements) ·
[6 Constraints](#6-constraints) ·
[7 Verification](#7-verification) ·
[Appendix A Test data](#appendix-a--test-data-generator)

## 1. Introduction

### Scope

**In scope:** analyzing one or more log files in the format of section 2; text and JSON reports;
concurrent processing controlled by the user; built-in profiling output.

**Out of scope:** following a growing file (follow mode) in any form, other log formats,
compressed input, reading from stdin or the network, persistent storage, and Windows support.

### Definitions

| Term | Meaning |
|---|---|
| Line | A sequence of bytes ended by `\n`, or by end of file for the last line. |
| Parsed line | A line that satisfies every rule in section 2. |
| Malformed line | A line that breaks at least one rule. It is counted and reported, never fatal. |
| Report | The statistics for every line processed in one run. |
| Worker | A unit of concurrent processing. `--jobs` sets the maximum number. |
| Reference machine | An Apple M4 Pro MacBook Pro on AC power. Performance targets in section 5 are stated for it. |

### Data sets

All data comes from `gen_logs.py` (Appendix A), which makes about 1% of lines malformed.

| ID | Contents | How to create it |
|---|---|---|
| D1 | 20,000 lines | `python3 gen_logs.py 20000 > data/access.log` |
| D2 | Five files of 1,000,000 lines (~58 MB each) | `python3 gen_logs.py 1000000 > data/big-N.log`, for N = 1..5 |
| D3 | One file of 5,000,000 lines (~290 MB) | `python3 gen_logs.py 5000000 > data/huge.log` |

## 2. Input format

Each line has six fields separated by runs of whitespace (spaces or tabs). Fields are checked left
to right; the first rule broken decides the line's error kind.

```
<timestamp> <method> <path> <status> <latency>ms <ip>
2026-11-16T22:56:01Z GET /api/users 200 34ms 10.0.2.220
```

| Field | Rule | Error kind if broken |
|---|---|---|
| timestamp | Any non-empty token. Kept as text, not validated. | missing field |
| method | Exactly one of `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS` (case-sensitive). | unknown method |
| path | Any non-empty token. | missing field |
| status | Decimal digits only (no sign), value 100 to 599. | invalid status |
| latency | Decimal digits only, followed by `ms`, value at most 4,294,967,295. | invalid latency |
| ip | A valid IPv4 or IPv6 address in standard text form, as accepted by Go's `netip.ParseAddr`. | invalid ip |

- A line that is empty or contains only whitespace has error kind **empty line**.
- A line with fewer than six fields has error kind **missing field**, naming the first absent field.
- A line with more than six fields has error kind **unexpected field**, quoting the first extra token.

### Status classes

| Status | 100–199 | 200–299 | 300–399 | 400–499 | 500–599 |
|---|---|---|---|---|---|
| Class | informational | success | redirect | client error | server error |

## 3. Functional requirements

### 3.1 Invocation and input

#### FR-1 · Command line

`loglens [flags] FILE...` analyzes the given files and writes one report to stdout. Flags come
before file arguments; accepting flags after files is optional.

| Flag | Default | Meaning |
|---|---|---|
| `--format text\|json` | `text` | Report format (FR-12, FR-13) |
| `-n`, `--top N` | 10 | Rows in each top-N list; 0 omits the lists |
| `--errors N` | 5 | Malformed lines listed; 0 lists none (the total is still shown) |
| `--jobs N` | usable CPUs | Maximum workers; 1 means no concurrent processing |
| `--max-errors N` | 0 (no limit) | Stop after N malformed lines (FR-16) |
| `--cpuprofile FILE` | | Write a CPU profile (FR-24) |
| `--memprofile FILE` | | Write a heap profile (FR-24) |
| `--trace FILE` | | Write an execution trace (FR-24) |
| `-h`, `--help` | | Print usage and exit 0 |

Acceptance

- [ ] `loglens -h` lists every flag and exits with 0.
- [ ] An unknown flag, an invalid value (for example `--jobs 0`, `--format xml`, `--top -1`) or
  no file arguments prints a usage message on stderr and exits with 2.

#### FR-2 · Reading files

Files are read as streams; the tool does not need the whole file in memory. Lines up to 1 MiB
(1,048,576 bytes, excluding the newline) are supported. A longer line is a runtime failure: the
tool reports the file and line number on stderr and exits with 1. A last line without a trailing
newline is still a line.

Acceptance

- [ ] A file containing a 1 MiB line is analyzed normally.
- [ ] A file containing a 2 MiB line fails with a message naming the file and line, and exit code 1.
- [ ] A file whose last line has no trailing newline counts and parses that line.

### 3.2 Parsing

#### FR-3 · Line parsing

Every line is parsed according to section 2.

Acceptance

- [ ] About 99% of D1 parses.
- [ ] Status values 99 and 600 are malformed; 100 and 599 are valid.
- [ ] `+200` as a status and `get` as a method are malformed.

#### FR-4 · Error classification

Every malformed line has exactly one error kind from section 2, decided by the first rule broken,
and a message in the form given in section 4.

Acceptance

- [ ] Each malformed line in Appendix A produces the error kind and message listed there.

### 3.3 Statistics

#### FR-5 · Line counts

The report states the total number of lines, parsed lines and malformed lines. Total = parsed +
malformed.

Acceptance

- [ ] The counts for D1 add up, and the total matches `wc -l` (plus one if the file lacks a
  trailing newline).

#### FR-6 · Status classes

The report gives the number of parsed lines in each status class and its percentage of parsed
lines, with one decimal place. Percentages are 0.0 when nothing parsed.

Acceptance

- [ ] The five class counts add up to the parsed count.

#### FR-7 · Latency percentiles

The report gives the exact p50, p95 and p99 latency over every parsed line in the report, using the
nearest-rank method: sort the latencies ascending; the p-th percentile is the value at 1-based rank
⌈p/100 × n⌉. With no parsed lines the percentiles are absent.

Acceptance

- [ ] For latencies 1 to 100 ms the result is p50 = 50, p95 = 95, p99 = 99.
- [ ] For a single parsed line all three equal its latency.
- [ ] With no parsed lines, text output shows `n/a` and JSON shows `null`.

#### FR-8 · Top IP addresses

The report lists the N IP addresses with the most parsed lines (N from `--top`), ordered by count
descending, then by address ascending (IPv4 before IPv6, then numerically).

Acceptance

- [ ] Results match `awk '{print $6}' data/access.log | sort | uniq -c | sort -rn`, apart from
  differences explained by malformed lines.
- [ ] A test with tied counts shows the address order.

#### FR-9 · Slowest endpoints

The report lists the N paths with the highest average latency, ordered by average descending, then
by path ascending (byte order). Each row shows the average, rounded to the nearest millisecond with
halves rounded up, and the number of parsed lines for that path.

Acceptance

- [ ] On D1, `/api/reports/export` is first.
- [ ] A test with tied averages shows the path order.

#### FR-10 · Malformed line list

The report lists the first N malformed lines (N from `--errors`), each with file name, line number
and message. "First" means files in command-line order, then ascending line number.

Acceptance

- [ ] The listed lines match the first malformed lines found by scanning the files in order by hand.

### 3.4 Output

#### FR-11 · Output streams

Reports go to stdout. Error messages and warnings go to stderr.

#### FR-12 · Text report

The text report follows the layout in section 4. Section order, labels and number formats are
normative; column widths and spacing are not.

Acceptance

- [ ] Output for D1 has every section of the sample, in order.

#### FR-13 · JSON report

The JSON report is a single JSON object with the keys shown in section 4. Every key is always
present: absent percentiles are `null` and empty lists are `[]`. Whitespace is not normative.

Acceptance

- [ ] `loglens --format json data/access.log | jq .` succeeds, and every key in the sample is
  present.

#### FR-14 · Several files

Several files produce one combined report. The text header lists the files, comma-separated, in
command-line order; JSON lists them in `files`.

Acceptance

- [ ] The report for D1 D1 has exactly twice the counts of the report for D1, the same
  percentiles, and errors listed from the first copy first.

#### FR-15 · Failures while analyzing

If a file can't be opened or read, the tool reports it on stderr, analyzes the remaining files,
prints the report for what it could read, and exits with 1. If writing the report fails, the tool
reports it on stderr and exits with 1.

Acceptance

- [ ] `loglens data/access.log nope.log` prints a report for D1, names `nope.log` on stderr and
  exits with 1.
- [ ] On Linux, `loglens data/access.log > /dev/full` exits with 1 and reports the write error.

### 3.5 Limits and interruption

#### FR-16 · Error limit

With `--max-errors N` (N > 0), once N malformed lines have been found, the tool stops processing
all files, prints no report, writes `too many malformed lines (limit N)` to stderr and exits with 1.

Acceptance

- [ ] `loglens --max-errors 10` on D2 exits with 1 and prints nothing to stdout.

#### FR-17 · Interruption

On SIGINT or SIGTERM during analysis, the tool stops processing, prints no report, writes
`interrupted` to stderr, and exits with 130 (SIGINT) or 143 (SIGTERM).

Acceptance

- [ ] Pressing Ctrl-C during a run over D2 exits with 130; `kill -TERM` exits with 143.

### 3.6 Follow mode (removed)

FR-18 to FR-23 covered follow mode, which is out of scope. The IDs are not reused.

### 3.7 Diagnostics

#### FR-24 · Profiles and traces

`--cpuprofile FILE` writes a CPU profile of the run, `--memprofile FILE` writes a heap profile
taken at the end of the run, and `--trace FILE` writes a Go execution trace.

Acceptance

- [ ] `go tool pprof` opens both profiles and `go tool trace` opens the trace.

## 4. Output formats

### Text report

Sample from D1 with `--top 5`. Numbers vary with the generated data.

```
loglens report: data/access.log
lines: 20000  parsed: 19776  failed: 224

status classes
  informational      0    0.0%
  success        14088   71.2%
  redirect        2873   14.5%
  client error    1952    9.9%
  server error     863    4.4%

latency  p50 40ms  p95 510ms  p99 1150ms

top 5 ips
  10.0.3.9      835
  10.0.0.252    439
  10.0.3.220    434

slowest 5 endpoints (average)
  /api/reports/export   454ms   3271 requests
  /api/products          46ms   3370 requests
  /api/login             46ms   3333 requests

malformed lines (224 total, first 5)
  data/access.log:45: unknown method "FETCH"
  data/access.log:214: invalid status "abc"
  data/access.log:257: invalid latency "fast"
```

### JSON report

Shortened to one element per list:

```json
{
  "files": ["data/access.log"],
  "lines": 20000, "parsed": 19776, "failed": 224,
  "status_classes": { "informational": 0, "success": 14088, "redirect": 2873,
                      "client_error": 1952, "server_error": 863 },
  "latency_ms": { "p50": 40, "p95": 510, "p99": 1150 },
  "top_ips": [ { "ip": "10.0.3.9", "count": 835 } ],
  "slowest_endpoints": [ { "path": "/api/reports/export", "avg_ms": 454, "count": 3271 } ],
  "malformed": [ { "file": "data/access.log", "line": 45,
                   "message": "unknown method \"FETCH\"" } ]
}
```

### Error messages

In reports, each message is prefixed with `file:line: `.

| Error kind | Message (example) |
|---|---|
| empty line | `empty line` |
| missing field | `missing field: latency` |
| unexpected field | `unexpected extra field "x"` |
| unknown method | `unknown method "FETCH"` |
| invalid status | `invalid status "abc"` |
| invalid latency | `invalid latency "fast"` |
| invalid ip | `invalid ip "10.0.0"` |

### Exit codes

| Code | Meaning |
|---|---|
| 0 | Success, including runs with malformed lines. |
| 1 | Runtime failure: a file couldn't be read, a line exceeded 1 MiB, output couldn't be written, or `--max-errors` was reached. |
| 2 | Usage error. |
| 130 | Interrupted by SIGINT. |
| 143 | Terminated by SIGTERM. |

## 5. Non-functional requirements

Measure performance and resource requirements as described in section 7. Wall times are for the
whole process, from start to exit.

### 5.1 Performance

#### NFR-1 · Single-worker throughput

With `--jobs 1`, the tool processes D2 at no less than 8 million lines per second on the reference
machine (at most 0.63 s for 5 million lines).

Acceptance

- [ ] Median of five runs of `loglens --jobs 1 data/big-*.log > /dev/null` is at most 0.63 s.

*Calibration:* on a 2.1 GHz Xeon vCPU, a straightforward first implementation processed about
2 million lines per second and an allocation-aware one about 5.8 million. On hardware other than
the reference machine, use 2× the `--jobs 1` throughput of your first working version as the
target.

#### NFR-2 · Scaling across files

With `--jobs 8`, wall time on D2 is no more than 25% of the `--jobs 1` wall time (a speedup of at
least 4×).

Acceptance

- [ ] Medians for `--jobs 1` and `--jobs 8` on D2 meet the ratio.

#### NFR-3 · Scaling within a file

With `--jobs 8`, wall time on D3 (a single file) is no more than 30% of the `--jobs 1` wall time (a
speedup of at least 3.3×).

Acceptance

- [ ] Medians for `--jobs 1` and `--jobs 8` on D3 meet the ratio.

#### NFR-4 · Allocations

Steady-state analysis averages at most one heap allocation per input line.

Acceptance

- [ ] An allocation-reporting benchmark (or runtime metrics) over D1 shows at most 20,000
  allocations attributable to analyzing it.

### 5.2 Resource usage

#### NFR-5 · Bounded memory

Peak memory does not grow with the number of lines or the size of the files; it may grow with the
number of distinct IP addresses, paths and latency values. For any `--jobs` from 1 to 16, peak
resident memory on D2 is at most 1.25× the peak on a single 1,000,000-line file, and never above
64 MiB.

Acceptance

- [ ] Peak RSS for `data/big-1.log` and for D2 meet both limits at `--jobs` 1, 8 and 16.

NFR-6 and NFR-7 covered follow mode, which is out of scope. The IDs are not reused.

### 5.3 Responsiveness

#### NFR-8 · Cancellation

After SIGINT or SIGTERM, the process exits within 500 ms, at any `--jobs` value.

Acceptance

- [ ] Run `--jobs 8` over `data/huge.log` listed 20 times (100 million lines), send SIGINT after
  one second, and the process ends within 500 ms. Repeat with `--jobs 1`.

#### NFR-9 · Error limit latency

Reaching `--max-errors` stops all processing within 500 ms.

Acceptance

- [ ] `loglens --max-errors 10 --jobs 8 data/big-*.log` completes in under 500 ms.

NFR-10 covered follow mode, which is out of scope. The ID is not reused.

### 5.4 Correctness and reliability

#### NFR-11 · Deterministic output

For the same input and flags, stdout is byte-for-byte identical across runs and for every `--jobs`
value.

Acceptance

- [ ] Twenty runs on D1 produce one distinct output.
- [ ] Runs on D2 with `--jobs` 1, 2, 3, 4, 8, 16 and 64 produce one distinct output.

#### NFR-12 · Robustness

No input can make the tool panic or crash.

Acceptance

- [ ] A fuzz test of the parsing path runs for at least five minutes without failures.
- [ ] A 50 MB file of random bytes is analyzed and the tool exits normally (0, or 1 if a line
  exceeds 1 MiB).

#### NFR-13 · Concurrency safety

The tool is free of data races.

Acceptance

- [ ] The full test suite, including tests with `--jobs` above 1, cancellation and `--max-errors`,
  passes under the race detector.

#### NFR-14 · No leaked goroutines

When an analysis finishes, fails, is cancelled or reaches `--max-errors`, no goroutine it started
is still running.

Acceptance

- [ ] Tests check for leftover goroutines on each of those paths.

### 5.5 Maintainability and testability

#### NFR-15 · Static checks

Code is gofmt-formatted, and `go vet` and `staticcheck` report nothing.

Acceptance

- [ ] `gofmt -l .` prints nothing; `go vet ./...` and `staticcheck ./...` are clean.

#### NFR-16 · Test coverage

Automated tests cover every error kind, the status range limits and class boundaries, the
percentile cases in FR-7, the tie rules in FR-8 and FR-9, both report formats, and every exit code.
Statement coverage is at least 80%, excluding package `main`.

Acceptance

- [ ] `go test -cover ./...` meets the threshold, and each listed case has a named test.

#### NFR-17 · Testable error kinds

Tests can tell error kinds apart without comparing message text.

Acceptance

- [ ] No test asserts an error kind by matching a message string.

#### NFR-18 · Documentation

Every exported identifier has a doc comment, and `loglens -h` is enough to use every feature.

Acceptance

- [ ] A linter that checks for missing doc comments on exported identifiers (for example revive's
  `exported` rule) reports nothing.

### 5.6 Portability

#### NFR-19 · Platforms

The tool builds and its tests pass on darwin/arm64 and linux/amd64. The default `--jobs` respects
CPU limits set for a container.

Acceptance

- [ ] `GOOS=linux GOARCH=amd64 go build ./...` succeeds on the reference machine, and tests pass
  in a Linux container.
- [ ] Inside a container limited to 2 CPUs (`docker run --cpus 2`), `loglens -h` shows a default
  `--jobs` of 2.

## 6. Constraints

| ID | Constraint |
|---|---|
| C-1 | Go 1.27. |
| C-2 | Runtime dependencies: the Go standard library and `golang.org/x` modules only. Test-only dependencies are unrestricted. |
| C-3 | No cgo and no `unsafe` package. |
| C-4 | One binary, named `loglens`. |

## 7. Verification

### Measurement protocol

- Use the reference machine on AC power, with other applications closed.
- Make one untimed run first so the files are in the operating system's cache.
- Report the median of five runs.
- Measure wall time with `time` or `hyperfine`, and peak resident memory with `/usr/bin/time -l`
  on macOS ("maximum resident set size") or `/usr/bin/time -v` on Linux.

### Results to record

Keep `NOTES.md` in the repository with, for every NFR you measure: the command, the raw numbers,
the median, the Go version and the machine. Re-measure after changes that could affect them.

## Appendix A · Test data generator

The generator is [`scripts/gen_logs.py`](../scripts/gen_logs.py). It needs only Python 3.

```sh
python3 gen_logs.py 20000 > data/access.log     # fixed number of lines
python3 gen_logs.py 1000000 > data/big-1.log    # ~58 MB
```

### Malformed lines it produces

| Line | Error kind | Message |
|---|---|---|
| *(empty line)* | empty line | `empty line` |
| `GARBAGE LINE` | unknown method | `unknown method "LINE"` |
| `2026-11-16T10:00:00Z GET /api/users abc 12ms 10.0.0.1` | invalid status | `invalid status "abc"` |
| `2026-11-16T10:00:00Z GET /api/users 200` | missing field | `missing field: latency` |
| `2026-11-16T10:00:00Z FETCH /api/users 200 5ms 10.0.0.1` | unknown method | `unknown method "FETCH"` |
| `2026-11-16T10:00:00Z GET /api/users 200 fast 10.0.0.1` | invalid latency | `invalid latency "fast"` |

`GARBAGE LINE` fails on the method, not on a missing field, because fields are checked left to
right and `LINE` sits in the method position.
