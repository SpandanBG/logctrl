# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev       # Run in development mode: go run main.go
make build     # Build binary to bin/out
make mock      # Run mock logger (requires Node.js): node ./tests/tools/mockLogger.js
make test      # gofmt check + go vet + CC-9 divider, GO-8 comment and doc-name checks + go test -race ./...
make cover     # Coverage check against COVER_MIN (excludes main and internal/; counted across packages), as run in CI
make hooks     # Enable the committed pre-commit hook (.githooks/), which runs `make test`
go test -bench=. ./reader  # Run benchmarks
```

## Testing (required)

Full rules: `CODING_STANDARDS.md` (Go, clean code, tests and TDD, CLI; each rule has an ID to cite) and `CONTRIBUTING.md` (workflow, docs to update, commits).

- **Every behaviour change goes through `/tdd`**: no production change without a failing test first, one vertical slice at a time.
- Test at seams (a package's public interface), not per file. Files with no behaviour of their own (constants, empty stubs, `main.go`) get no tests.
- When a test is added for code that already exists, prove it can fail: `scripts/prove-fails.sh <file> <old> <new> <go test args>` breaks the code, runs the tests and restores it (exit 0 = caught).
- Shared helpers live in `internal/`: `testutil.Equal`, `testutil.Runtime` (drives a `tea.Model` the way the tea runtime does, answering `tea.WindowSize()`), `testutil.Colors` (a lipgloss renderer that always outputs ANSI colors, so styled output can be compared against literals), `logfeed` (a real `reader.Stream` fed over a pipe, for UI and Executor tests, with the doubles and Lines they share: `Gated`, `Numbered`) and `logview` (reads the Log View's rows out of a rendered view).
- Tests are table-driven where cases vary only by data; names use the terms in `GLOSSARY.md`.
- CI (`.github/workflows/test.yml`) runs `make test` and `make cover`. Raise `COVER_MIN` in the `Makefile` as coverage grows; never lower it to make a change pass.

## Architecture

`logctrl` is a terminal UI log viewer: a parent process puts `/dev/tty` in raw mode and re-execs itself as a child, which reads the log stream from stdin and runs the BubbleTea UI. Read [`docs/architecture.md`](docs/architecture.md) before changing a package's role, a key interface (`reader.Stream`, `cmd.Stream`, `headless.Logs`) or a key binding; it holds the data flow, each package's role, the interfaces and the built-in keys, and is updated with them.

Keys are dispatched by Action from the Keymap (ADR 0003): to bind a key in the UI, add an Action to `config/keymap.json`.

## Agent skills

### Issue tracker

Issues live in GitHub Issues for SpandanBG/logctrl, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
