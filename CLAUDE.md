# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is assay?

A CLI tool that estimates the value of a codebase using COCOMO II cost modeling. It analyzes source code for SLOC, complexity, test coverage, dependency health, git history, and code duplication, then produces a composite score and dollar-value estimate.

## Build & Run

```bash
go build -o assay ./cmd/assay     # build binary
go run ./cmd/assay [path]          # run directly
go run ./cmd/assay --format json   # JSON output
go run ./cmd/assay --verbose       # per-file breakdown
go run ./cmd/assay --rate 200      # custom hourly rate (default: $150)
go run ./cmd/assay --exclude "*.generated.go,vendor/*"
```

There are no tests yet. The module is `github.com/ao/assay` with Go 1.22.

## Architecture

The pipeline is: **walk files -> analyze (parallel) -> cost model -> render output**.

- `cmd/assay/main.go` — CLI entry point using cobra. Orchestrates the pipeline.
- `internal/walker/` — Directory traversal respecting `.gitignore` and `--exclude` patterns. Maps file extensions to language names via `langMap`.
- `internal/analyzer/` — All analysis logic, run concurrently via a worker pool:
  - `types.go` — `FileInfo`, `Metrics`, `FileStat` structs shared across analyzers
  - `analyze.go` — Orchestrator that fans out per-file analysis (SLOC + complexity) to workers, then runs deps/git/duplication in parallel
  - `sloc.go` — SLOC counting with per-language comment syntax awareness
  - `complexity.go` — Cyclomatic complexity approximation via decision-point keyword counting
  - `deps.go` — Dependency manifest detection (go.mod, package.json, requirements.txt, Cargo.toml) and counting
  - `git.go` — Git history analysis using go-git (commits, contributors, repo age)
  - `duplication.go` — Line-level duplication detection using SHA-256 hashing (flags lines appearing 3+ times)
- `internal/model/cocomo.go` — COCOMO II Basic cost estimation with multipliers for low tests, high duplication, missing lockfiles, stale repos. Also computes weighted dimension scores (composite out of 100).
- `internal/report/` — Output rendering:
  - `table.go` — Human-readable table with Unicode box-drawing
  - `json.go` — Structured JSON output

## Key design details

- The `Metrics` struct in `analyzer/types.go` is the central data structure flowing through the entire pipeline.
- Per-file analysis (SLOC, complexity) uses a channel-based worker pool sized to `runtime.NumCPU()`.
- Dependency, git, and duplication analyses run as three parallel goroutines after per-file analysis completes.
- Cost multipliers adjust the COCOMO base cost up/down based on code health signals; the range output is base * 0.8 to base * 1.2.
- Scores are weighted: Size/Effort 25%, Code Quality 25%, Test Coverage 20%, Dep Health 15%, Git Activity 15%.
