# loglens

A command-line tool, written in Go, that analyzes access logs and reports traffic, errors and
latency. It has a batch mode for one or more files and a follow mode for a single growing log.

> **Status:** work in progress. This is a learning project for Go; the behavior below is what the
> tool is being built to do, as specified in a software requirements specification (SRS v4.0)
> that is kept out of this repository. Requirement IDs like FR-7 and NFR-4 refer to that
> document.

## Usage

```sh
loglens [flags] FILE...
```

| Flag | Default | Meaning |
|---|---|---|
| `--format text\|json` | `text` | Report format |
| `-n`, `--top N` | `10` | Rows in each top-N list; `0` omits the lists |
| `--errors N` | `5` | Malformed lines listed; `0` lists none (the total is still shown) |
| `--jobs N` | usable CPUs | Maximum workers; `1` means no concurrent processing |
| `--max-errors N` | `0` (no limit) | Stop after N malformed lines |
| `--follow` | off | Follow one growing file |
| `--interval DURATION` | `5s` | Follow mode: time between reports (at least `100ms`) |
| `--log-level LEVEL` | `info` | Follow mode: `debug`, `info`, `warn` or `error` |
| `--cpuprofile FILE` | | Write a CPU profile |
| `--memprofile FILE` | | Write a heap profile |
| `--trace FILE` | | Write an execution trace |
| `-h`, `--help` | | Print usage and exit |

Examples:

```sh
loglens data/access.log                         # text report
loglens --format json --top 5 data/big-*.log    # one combined JSON report over several files
loglens --follow --interval 1s data/live.log    # a report every second until Ctrl-C
```

Reports go to stdout; errors, warnings and follow-mode logs go to stderr.

## Input format

One request per line, six fields separated by spaces or tabs:

```
<timestamp> <method> <path> <status> <latency>ms <ip>
2026-11-16T22:56:01Z GET /api/users 200 34ms 10.0.2.220
```

| Field | Rule |
|---|---|
| timestamp | Any non-empty token (not validated) |
| method | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD` or `OPTIONS` (case-sensitive) |
| path | Any non-empty token |
| status | Digits only, 100 to 599 |
| latency | Digits followed by `ms`, at most 4,294,967,295 |
| ip | IPv4 or IPv6 address |

Lines that break a rule are counted and listed as malformed, never fatal. Fields are checked left
to right and the first broken rule decides the error:

| Error kind | Example message |
|---|---|
| empty line | `empty line` |
| missing field | `missing field: latency` |
| unexpected field | `unexpected extra field "x"` |
| unknown method | `unknown method "FETCH"` |
| invalid status | `invalid status "abc"` |
| invalid latency | `invalid latency "fast"` |
| invalid ip | `invalid ip "10.0.0"` |

Lines may be up to 1 MiB long.

## Report

```
loglens report: data/access.log
lines: 20000  parsed: 19776  failed: 224

status classes
  informational       0    0.0%
  success         14088   71.2%
  redirect         2873   14.5%
  client error     1952    9.9%
  server error      863    4.4%

latency  p50 40ms  p95 510ms  p99 1150ms

top 5 ips
  10.0.3.9        835
  10.0.0.252      439
  10.0.3.220      434

slowest 5 endpoints (average)
  /api/reports/export    454ms    3271 requests
  /api/products           46ms    3370 requests
  /api/login              46ms    3333 requests

malformed lines (224 total, first 5)
  data/access.log:45: unknown method "FETCH"
  data/access.log:214: invalid status "abc"
  data/access.log:257: invalid latency "fast"
```

- Percentiles are exact, using the nearest-rank method.
- Top IPs are ordered by count, then address; slowest endpoints by average latency, then path.
- With `--format json` the same report is one JSON object with the keys `files`, `lines`,
  `parsed`, `failed`, `status_classes`, `latency_ms`, `top_ips`, `slowest_endpoints` and
  `malformed`. In follow mode, each report is one compact JSON object per line.
- Output is identical for the same input, whatever `--jobs` is.

## Follow mode

`loglens --follow FILE` starts at the current end of the file and writes a report covering every
line since it started, every `--interval`. If the file is truncated it logs a warning and reads
from the start again. Ctrl-C (or SIGTERM) writes a final report and exits.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success, including runs with malformed lines; follow mode stopped by a signal |
| 1 | A file couldn't be read, a line exceeded 1 MiB, output couldn't be written, or `--max-errors` was reached |
| 2 | Usage error |
| 130 | Batch mode interrupted by SIGINT |
| 143 | Batch mode terminated by SIGTERM |

## Development

Requires Go 1.27. Runtime code uses only the standard library and `golang.org/x` modules, with no
cgo and no `unsafe`.

```sh
go build -o loglens ./cmd/loglens   # build
go test ./...                       # tests
go test -race ./...                 # race detector
go test -cover ./...                # coverage (target: 80%, excluding package main)
gofmt -l . && go vet ./... && staticcheck ./... && revive ./...   # static checks
```

### Test data

Test logs come from `scripts/gen_logs.py` (from SRS Appendix A); about 1% of generated lines
are malformed.

```sh
python3 scripts/gen_logs.py 20000 > data/access.log                                     # D1
for n in 1 2 3 4 5; do python3 scripts/gen_logs.py 1000000 > data/big-$n.log; done     # D2, ~58 MB each
python3 scripts/gen_logs.py 5000000 > data/huge.log                                     # D3, ~290 MB
python3 scripts/gen_logs.py 0 --rate 1000 >> data/live.log                              # D4, endless
```

`data/` is not committed.

### Performance

Targets on an Apple M4 Pro include at least 8 million lines/s with `--jobs 1`, a 4× speedup at
`--jobs 8` across files, at most one heap allocation per line, and under 64 MiB peak memory.
Measurements are recorded in `NOTES.md`.

## Layout

```
cmd/loglens/   the loglens binary (kept thin)
internal/      parsing, statistics, reporting, follow mode
scripts/       gen_logs.py
data/          generated test logs (not committed)
```
