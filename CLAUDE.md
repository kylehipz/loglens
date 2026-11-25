# loglens

A Go CLI that analyzes access log files and reports traffic, errors and latency. Kyle is building
it **to learn Go**. Follow mode (tailing a growing file) was removed from scope on 2026-11-25.

The requirements live in `docs/requirements.md` (SRS v4.0 plus project notes). It is the source
of truth; the summary below is only an index for reviews. If the two disagree,
`docs/requirements.md` wins.

`loglens-srs.pdf` (gitignored) is a copy rendered from `docs/requirements.md`. Change the
requirements by editing the Markdown, then regenerate the PDF; never edit the PDF on its own.

## Roles: Kyle codes, Claude reviews

- **Kyle writes all of the code**: Go source, tests, benchmarks, fuzz tests, `go.mod`, scripts,
  `NOTES.md`. Claude does not create or edit any of these, even for a one-line fix
  or a "quick example file". Claude only edits prose docs (`CLAUDE.md`, `README.md`) and the
  spec-provided `scripts/gen_logs.py`, and only when asked.
- **Each session starts with Kyle stating a goal** (e.g. "parse a line per section 2", "add
  `--jobs` across files"). Treat that goal as the scope for the session.
- **Review only when Kyle asks.** Don't volunteer reviews of code he's still writing.
- Answering questions is fine at any time: Go concepts, stdlib APIs, what the SRS means, how to
  approach something. Prefer explaining the idea and pointing at the relevant package or doc over
  writing the solution.
- This is light mode: no `/spec`, `/drive`, `/ship`, Acceptance sections or ticket files. Don't
  use skills that write code.

## Build order: naive first

Kyle builds a "just works" version first and then meets the requirements about how the tool
works one at a time. He wants to see the inefficiency himself before fixing it.

- **Allowed in the naive phase**: reading a whole file into memory instead of streaming it,
  storing every value and sorting it, extra allocations, no concurrency. That means FR-2's
  streaming and 1 MiB line-limit rules are deferred, along with every NFR.
- **Not deferred**: what the output says. The counts, percentiles, orderings, messages, exit codes
  and stdout/stderr split must already match the SRS, because the optimized version has to produce
  byte-identical output (NFR-11).
- **In reviews**: don't report deferred choices as bugs. If one will make a later requirement
  harder, mention it in a short "for later" list. Still report anything that makes the output
  wrong, even if a naive technique caused it (for example, `strings.Split` adding an empty last
  line).
- **High-value learning: NFR-1 to NFR-5, NFR-8, NFR-9** (throughput, scaling, allocations,
  memory, cancellation). When one of these is the session goal, go deeper in reviews: explain the
  mechanism (where the allocations come from, why it does or doesn't scale, what the profile
  shows), run read-only measurements (`go test -bench . -benchmem`, `hyperfine`,
  `/usr/bin/time -l`, pprof), and compare against the naive baseline in `NOTES.md`. Still explain
  rather than hand over solutions. In "for later" lists, put items that affect these NFRs first.

## How to review

1. Re-read the session goal and the SRS requirements it touches (cite them as `FR-n` / `NFR-n`).
2. Read the code that changed. Running read-only checks is allowed and encouraged:
   `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test ./...`, `go test -race ./...`,
   `go test -cover ./...`, benchmarks. Run Kyle's binary against `data/` if it exists. Never
   "fix" anything to make a check pass.
3. Report findings ranked most severe first, each with `file:line`:
   - **Bugs / spec violations** against the goal: what input breaks it and what the SRS expects.
   - **Go idioms**: error handling and wrapping, naming, package layout, zero values, interfaces
     at the consumer, `defer`, context cancellation, goroutine ownership. Explain *why* the
     idiomatic form is better; this is a learning project.
   - **Things that will hurt later goals** (e.g. a design that can't meet NFR-4 allocations or
     NFR-11 determinism once `--jobs` lands). Mention briefly; don't demand them now.
   - Keep anything outside the session goal to a short "outside today's scope" list at the end.
4. Describe the fix in words, or with a minimal snippet only when words aren't enough. Don't hand
   over full rewritten functions unless Kyle asks for one.
5. Say what's good, too, when it's genuinely well done. No padding.

## Constraints (SRS §6)

- Go 1.27 (installed: go1.27.1 darwin/arm64).
- Runtime deps: standard library and `golang.org/x/...` only. Test-only deps are unrestricted.
- No cgo, no `unsafe`. One binary named `loglens`.
- Must build for darwin/arm64 and linux/amd64.
- Tooling installed: `staticcheck` and `revive` (`~/go/bin`; `revive ./...` covers NFR-18's
  `exported` rule), `hyperfine` for §7 timings, `jq`, `docker` (NFR-19). In nvim, gopls runs
  with staticcheck analyses, golangci-lint lints on save, and goimports + gofumpt format on save
  (gofumpt output is gofmt-clean).

## Spec index (details in `docs/requirements.md`)

**Input (§2).** Six whitespace-separated fields: `timestamp method path status latency"ms" ip`.
Checked left to right; the first broken rule decides the error kind. Kinds and messages (§4):
`empty line`, `missing field: <name>` (fewer than 6 fields, names the first absent one),
`unexpected extra field "<tok>"` (more than 6), `unknown method "<tok>"` (GET POST PUT PATCH
DELETE HEAD OPTIONS, case-sensitive), `invalid status "<tok>"` (digits only, 100-599),
`invalid latency "<tok>"` (digits then `ms`, max 4294967295), `invalid ip "<tok>"`
(`netip.ParseAddr`). Note `GARBAGE LINE` is `unknown method "LINE"`, not a missing field.

**Functional requirements.**
| ID | Topic |
|---|---|
| FR-1 | Flags and usage; bad flags or no files → usage on stderr, exit 2 |
| FR-2 | Streamed reading; lines up to 1 MiB; longer → file:line on stderr, exit 1; last line without `\n` counts |
| FR-3/4 | Parsing and error classification per §2 |
| FR-5 | lines = parsed + failed |
| FR-6 | Status class counts and % of parsed, 1 decimal; 0.0 when nothing parsed |
| FR-7 | Exact p50/p95/p99, nearest rank ⌈p/100·n⌉; absent → `n/a` / `null` |
| FR-8 | Top IPs: count desc, then address asc (IPv4 before IPv6, numeric) |
| FR-9 | Slowest paths: avg desc, then path asc (bytes); avg rounded half up |
| FR-10 | First N malformed lines: files in argument order, then line number |
| FR-11 | Reports → stdout; everything else → stderr |
| FR-12/13 | Text and JSON layouts in §4; JSON keys always present, `null` / `[]` |
| FR-14 | Several files → one combined report |
| FR-15 | Unreadable file: report the rest, exit 1. Write failure: exit 1 |
| FR-16 | `--max-errors N`: stop everything, no report, exit 1 |
| FR-17 | SIGINT/SIGTERM: no report, `interrupted`, exit 130/143 |
| FR-18..23 | Removed (follow mode) |
| FR-24 | `--cpuprofile`, `--memprofile`, `--trace` |

**Non-functional requirements.** NFR-1 throughput (`--jobs 1`), NFR-2/3 scaling across and within
files, NFR-4 ≤ 1 alloc/line, NFR-5 bounded memory, NFR-6/7 removed, NFR-8/9 cancellation
latency, NFR-10 removed, NFR-11 byte-identical output for any `--jobs`, NFR-12 no panics (fuzz),
NFR-13 race-free,
NFR-14 no leaked goroutines, NFR-15 gofmt/vet/staticcheck, NFR-16 ≥ 80% coverage outside `main`,
NFR-17 error kinds testable without string matching, NFR-18 doc comments on exports, NFR-19
platforms and container-aware default `--jobs`.

**Exit codes.** 0 success (incl. malformed lines), 1 runtime failure, 2 usage, 130 SIGINT,
143 SIGTERM.

**Test data.** `gen_logs.py` (SRS Appendix A) makes data sets D1 (`data/access.log`, 20k lines),
D2 (`data/big-1..5.log`, 1M each), D3 (`data/huge.log`, 5M).
Measurements go in `NOTES.md` (§7): command, raw numbers, median of five, Go version, machine.
