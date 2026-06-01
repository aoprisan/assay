<p align="center">
  <img src="docs/logo.svg" alt="assay" width="480">
</p>

<p align="center">
  <em>Estimate what a codebase is worth — in effort, schedule, and dollars.</em>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#usage">Usage</a> ·
  <a href="#what-it-measures">What it measures</a> ·
  <a href="#how-the-estimate-works">How it works</a> ·
  <a href="#architecture">Architecture</a>
</p>

---

**assay** is a single-binary CLI that points at any source tree and produces a
defensible estimate of its development value. It walks the files, measures size,
complexity, test coverage, dependency health, git history, and code duplication,
then runs the numbers through a [COCOMO II](https://en.wikipedia.org/wiki/COCOMO)
cost model to output a composite health score and a dollar-value range.

```
──────────────────────────────────────────────────
  assay — Codebase Value Estimator
  Path: .
──────────────────────────────────────────────────
  SLOC                 1,259  (Go 100%)
  Files                12
  Complexity           avg 23.5 / file
  Test Ratio           0%  (0 test, 12 source)
  Duplication          17.9%
  Dependencies         27  (go.mod ✓)
  Git Activity         4 commits, 3 contributors
  Repo Age             87 days
  Last Commit          87 days ago

  ┌──────────────────────┬──────────┐
  │ Dimension            │  Score   │
  ├──────────────────────┼──────────┤
  │ Size & Effort        │   77/100 │
  │ Code Quality         │   29/100 │
  │ Test Coverage        │    0/100 │
  │ Dependency Health    │   85/100 │
  │ Git Activity         │   70/100 │
  └──────────────────────┴──────────┘

  Composite Score        49 / 100

──────────────────────────────────────────────────
  Cost Estimate
──────────────────────────────────────────────────
  Hourly Rate            $150/hr
  Base Cost              $90,904
  Adjustment Factor      x1.93
  Adjusted Cost          $175,628
  Estimated Range        $140,502 – $210,753  (±20%, high confidence)
  Cost / SLOC            $139

  Effort                 7.3 person-months
  Schedule               6.4 months
  Team Size (avg)        1.1 developers

  Adjustment Details:
    ⚠ Very Low Test Coverage       x1.40  test ratio < 5%
    ⚠ High Duplication             x1.20  duplication > 15%
    ⚠ High Complexity              x1.15  avg complexity > 15 per file
──────────────────────────────────────────────────
```

## Install

Requires [Go 1.22+](https://go.dev/dl/).

```bash
# build a local binary
go build -o assay ./cmd/assay

# or install onto your PATH
go install github.com/ao/assay/cmd/assay@latest
```

## Usage

```bash
assay [path] [flags]
```

If no path is given, assay analyzes the current directory.

```bash
assay                                   # analyze the current directory
assay ./path/to/project                 # analyze a specific tree
assay --format json                     # machine-readable output
assay --verbose                         # add a per-file breakdown
assay --rate 200                         # use a $200/hr developer rate
assay --exclude "*.generated.go,vendor/*"  # skip matching files
```

You can also run without building:

```bash
go run ./cmd/assay [path]
```

### Flags

| Flag        | Default | Description                                          |
| ----------- | ------- | ---------------------------------------------------- |
| `--rate`    | `150`   | Hourly developer rate in USD.                        |
| `--format`  | `table` | Output format: `table` or `json`.                    |
| `--exclude` | `""`    | Comma-separated glob patterns to skip.               |
| `--verbose` | `false` | Show a per-file SLOC / complexity / language table.  |

The walker automatically respects `.gitignore` and skips common non-source
directories (`node_modules`, `vendor`, `dist`, `build`, `target`,
`__pycache__`, `.git`, and hidden directories).

## What it measures

| Dimension             | Signal                                                                 |
| --------------------- | ---------------------------------------------------------------------- |
| **Size & Effort**     | Source lines of code (SLOC), with a per-language breakdown.            |
| **Code Quality**      | Cyclomatic complexity (decision-point approximation) + duplication.    |
| **Test Coverage**     | Ratio of test files/lines to source as a coverage proxy.               |
| **Dependency Health** | Manifest detection, dependency count, and lockfile presence.          |
| **Git Activity**      | Commit count, contributors, repo age, and recency of the last commit.  |

Languages are detected by file extension and include Go, JavaScript/TypeScript,
Python, Rust, Java, C/C++, C#, Ruby, PHP, Swift, Kotlin, Scala, Shell, and more.

Dependency manifests recognized: `go.mod`, `package.json`, `requirements.txt`,
and `Cargo.toml`.

## How the estimate works

assay uses **COCOMO II Basic** as its foundation:

```
effort (person-months) = 2.94 × KSLOC^1.10
schedule (months)      = 3.67 × effort^0.28
```

Effort is first weighted by the language mix (e.g. C/C++ costs more per line than
Shell), converted to a base cost at the configured hourly rate (160 working hours
per month), then adjusted by a set of **quality and risk multipliers**:

- **Test coverage** — thin coverage raises cost; strong coverage lowers it.
- **Duplication** — high duplication raises cost.
- **Complexity** — high average complexity per file raises cost.
- **Dependencies** — a missing lockfile or heavy dependency count raises cost.
- **Repository health** — stale repos are discounted; mature multi-contributor
  projects get a small confidence bonus.

The final figure is reported as a range. The width of that range (±20% / ±35% /
±50%) reflects a **confidence level** computed from how many independent signals
were available (SLOC, git history, dependency data, test data, duplication).

The **composite score** (0–100) is a weighted blend of the five dimensions:

| Dimension         | Weight |
| ----------------- | ------ |
| Size & Effort     | 25%    |
| Code Quality      | 25%    |
| Test Coverage     | 20%    |
| Dependency Health | 15%    |
| Git Activity      | 15%    |

> [!NOTE]
> assay produces an *estimate*, not an appraisal. COCOMO is a parametric model
> built on historical project data; treat the output as a directional signal for
> comparison and conversation, not a precise valuation.

## Architecture

The pipeline is **walk → analyze (parallel) → cost model → render**.

```
cmd/assay/main.go        CLI entry point (cobra); orchestrates the pipeline
internal/walker/         Directory traversal, .gitignore + exclude handling, language mapping
internal/analyzer/       Concurrent analysis (worker pool sized to NumCPU)
  ├─ types.go            Shared Metrics / FileStat structs
  ├─ analyze.go          Fan-out orchestrator
  ├─ sloc.go             SLOC counting with per-language comment awareness
  ├─ complexity.go       Cyclomatic complexity approximation
  ├─ deps.go             Dependency manifest detection & counting
  ├─ git.go              Git history analysis (go-git)
  └─ duplication.go      Line-level duplication via SHA-256 hashing
internal/model/cocomo.go COCOMO II estimation, multipliers, dimension scores
internal/report/         Output rendering (table.go, json.go)
```

Per-file analysis (SLOC + complexity) runs through a channel-based worker pool;
dependency, git, and duplication analyses then run as three parallel goroutines.
The `Metrics` struct in `internal/analyzer/types.go` is the central data
structure that flows through the whole pipeline.

## License

See repository for license details.
