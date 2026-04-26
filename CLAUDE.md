# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev       # Run in development mode: go run main.go
make build     # Build binary to bin/out
make mock      # Run mock logger (requires Node.js): node ./tests/tools/mockLogger.js
go test ./reader           # Run unit tests (buffer package has table-driven tests + benchmarks)
go test -bench=. ./reader  # Run benchmarks
```

## Architecture

`logctrl` is a terminal UI log viewer that reads log streams from stdin and displays them interactively. It uses a **parent-child process model**:

- **Parent** (`main.go`): Opens `/dev/tty` in raw mode, creates a pipe, re-execs itself as a child with `logctrl_child=<fd>` env var, then multiplexes I/O between PTY master/slave and stdin. Handles `SIGWINCH` (resize) and `SIGTERM` (kills process group on exit).
- **Child** (`main.go → startChildProcess`): Detects `logctrl_child` env var, reads logs from that file descriptor via `StreamV2`, and launches the BubbleTea UI.

### Data Flow

```
stdin (log producer) ──→ anonymous pipe ──→ StreamV2 ──→ circular Buffer ──→ LogView (viewport)
                                       └──→ temp file (logCtrl_logFile.txt in os.TempDir())

/dev/tty ──→ PTY slave (child) ──→ BubbleTea event loop
PTY master ──→ stdout (rendered UI)
```

### Key Packages

| Package | Role |
|---|---|
| `reader/` | `StreamV2` multiplexes the log feed into temp file + circular buffer; `Buffer` is a ring buffer with dynamic resize |
| `ui/` | BubbleTea layout model orchestrating three components: Toolbar, LogView, Prompt |
| `ui/components/` | `LogView` (viewport + StreamV2 poll), `Toolbar` (keybind help), `Prompt` (togglable textarea) |
| `ui/utils/` | Responsive sizing system (`Ratio`, `Fixed`, `Modifier` types) and ANSI color constants |
| `utils/` | `SIGWINCH` handler with callback pattern |

### Key Interfaces

**`reader.StreamV2`** — the bridge between the pipe and the UI:
```go
Start(chan bool)    // begin consuming from pipe
SetBufferSize(int)  // resize display buffer on terminal resize
GetLive() string    // get current buffer contents for rendering
Close()
```

**`reader.Buffer`**:
```go
Push(string)           // add log line
Resize(int)            // dynamic resize (preserves recent lines)
Stringify(string) string
```

All UI components implement `tea.Model` (`Init`, `Update`, `View`).

### UI Key Bindings

| Key | Action |
|---|---|
| `Tab` | Toggle prompt panel |
| `c` | Clear log view |
| `q` / `^C` / `^D` | Quit |

### Environment Variables

- `logctrl_child` — integer file descriptor; set by parent on re-exec to signal child mode.
