# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev       # Run in development mode: go run main.go
make build     # Build binary to bin/out
make mock      # Run mock logger (requires Node.js): node ./tests/tools/mockLogger.js
make test      # go vet + go test -race ./...
make cover     # Coverage check against COVER_MIN (excludes main and internal/), as run in CI
make hooks     # Enable the committed pre-commit hook (.githooks/), which runs `make test`
go test -bench=. ./reader  # Run benchmarks
```

## Testing (required)

- **Every behaviour change goes through `/tdd`**: no production change without a failing test first, one vertical slice at a time.
- Test at seams (a package's public interface), not per file. Files with no behaviour of their own (constants, empty stubs, `main.go`) get no tests.
- When a test is added for code that already exists, prove it can fail: break the code on purpose, watch the test fail, undo the break.
- Shared helpers live in `internal/`: `testutil.Equal`, `testutil.Runtime` (drives a `tea.Model` the way the tea runtime does, answering `tea.WindowSize()`), and `fakestream.Stream` (an in-memory `reader.Stream` for UI tests).
- Tests are table-driven where cases vary only by data; names use the terms in `GLOSSARY.md`.
- CI (`.github/workflows/test.yml`) runs `make test` and `make cover`. Raise `COVER_MIN` in the `Makefile` as coverage grows; never lower it to make a change pass.

## Architecture

`logctrl` is a terminal UI log viewer that reads log streams from stdin and displays them interactively. It uses a **parent-child process model**:

- **Parent** (`main.go`): Opens `/dev/tty` in raw mode, creates a pipe, re-execs itself as a child with `logctrl_child=<fd>` env var, then multiplexes I/O between PTY master/slave and stdin. Handles `SIGWINCH` (resize) and `SIGTERM` (kills process group on exit).
- **Child** (`main.go → startChildProcess`): Detects `logctrl_child` env var, reads logs from that file descriptor via `Stream`, and launches the BubbleTea UI.

### Data Flow

```
stdin (log producer) ──→ anonymous pipe ──→ Stream ──→ LiveBuffer ──→ Window ──→ LogView (viewport)
                                       └──→ temp file (logCtrl_logFile.txt in os.TempDir())

/dev/tty ──→ PTY slave (child) ──→ BubbleTea event loop
PTY master ──→ stdout (rendered UI)
```

### Key Packages

| Package | Role |
|---|---|
| `reader/` | `Stream` tees the Log Feed into the Log File and a `LiveBuffer` (bounded ring addressed by Line Number); a `Window` (two Line Numbers + Following/Pinned) selects what is shown. `Buffer` is the older ring buffer, now unused |
| `ui/` | BubbleTea layout model orchestrating three components: Toolbar, LogView, Prompt |
| `ui/components/` | `LogView` (viewport + Stream poll), `Toolbar` (keybind help), `Prompt` (togglable textarea) |
| `ui/utils/` | Responsive sizing system (`Ratio`, `Fixed`, `Modifier` types) and ANSI color constants |
| `utils/` | `SIGWINCH` handler with callback pattern |
| `internal/` | Test-only helpers (`testutil`, `fakestream`) |

Domain terms are defined in `GLOSSARY.md`; design decisions in `docs/adr/`.

### Key Interfaces

**`reader.Stream`** — the bridge between the pipe and the UI:
```go
Start(chan bool)    // begin consuming from pipe; notified after each line is held
SetBufferSize(int)  // set the Window height (held lines are kept)
GetWindow() string  // lines in the Window, for rendering
MoveToEnd()         // Window follows the newest lines
MoveBy(int) int     // scroll the Window; returns lines actually moved
Close()
```

All UI components implement `tea.Model` (`Init`, `Update`, `View`).

### UI Key Bindings

| Key | Action |
|---|---|
| `Tab` | Open prompt panel |
| `Esc` | Close prompt panel (while open) |
| `q` / `^C` / `^D` | Quit (while prompt closed) |

### Environment Variables

- `logctrl_child` — integer file descriptor; set by parent on re-exec to signal child mode.
