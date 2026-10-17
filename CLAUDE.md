# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev       # Run in development mode: go run main.go
make build     # Build binary to bin/out
make mock      # Run mock logger (requires Node.js): node ./tests/tools/mockLogger.js
make test      # go vet + go test -race ./...
make cover     # Coverage check against COVER_MIN (excludes main and internal/; counted across packages), as run in CI
make hooks     # Enable the committed pre-commit hook (.githooks/), which runs `make test`
go test -bench=. ./reader  # Run benchmarks
```

## Testing (required)

- **Every behaviour change goes through `/tdd`**: no production change without a failing test first, one vertical slice at a time.
- Test at seams (a package's public interface), not per file. Files with no behaviour of their own (constants, empty stubs, `main.go`) get no tests.
- When a test is added for code that already exists, prove it can fail: break the code on purpose, watch the test fail, undo the break.
- Shared helpers live in `internal/`: `testutil.Equal`, `testutil.Runtime` (drives a `tea.Model` the way the tea runtime does, answering `tea.WindowSize()`), and `fakestream.Stream` (an in-memory `cmd.Stream` for UI and Executor tests).
- Tests are table-driven where cases vary only by data; names use the terms in `GLOSSARY.md`.
- CI (`.github/workflows/test.yml`) runs `make test` and `make cover`. Raise `COVER_MIN` in the `Makefile` as coverage grows; never lower it to make a change pass.

## Architecture

`logctrl` is a terminal UI log viewer that reads log streams from stdin and displays them interactively. It uses a **parent-child process model**:

- **Parent** (`main.go`): Opens `/dev/tty` in raw mode, creates a pipe, re-execs itself as a child with `logctrl_child=<fd>` env var, then multiplexes I/O between PTY master/slave and stdin. Handles `SIGWINCH` (resize) and `SIGTERM` (kills process group on exit).
- **Child** (`main.go → startChildProcess`): Detects `logctrl_child` env var, reads logs from that file descriptor via `Stream`, wraps it in the `cmd.Executor`, and launches the BubbleTea UI.

### Data Flow

```
stdin (log producer) ──→ anonymous pipe ──→ Stream ──→ History (LiveBuffer + Chunks) ──→ Window ──→ Executor (Passes) ──→ LogView (viewport)
                                       └──→ temp file (logCtrl_logFile.txt in os.TempDir()) ──→ Chunks

/dev/tty ──→ PTY slave (child) ──→ BubbleTea event loop
PTY master ──→ stdout (rendered UI)
```

### Key Packages

| Package | Role |
|---|---|
| `reader/` | `Stream` tees the Log Feed into the Log File and a `History`: a `LiveBuffer` (bounded ring addressed by Line Number) plus older lines loaded from the Log File in Chunks via a sparse Line Index (ADR 0004). A `Window` (two Line Numbers + Following/Pinned) selects what is shown, back to Line Number 0. Each held line is a `Line` (its text, and its Matches once a Pass has run), built once on push or Chunk load; `Window.Lines` and `Stream.Lines` return `[]Line` |
| `cmd/` | The `Executor` wraps a `reader.Stream` and is the `cmd.Stream` the UI uses: on each read it runs every Pass (`search.go`: marks each Line's Matches for the Search regex) over copies of the Window's Lines, then joins their text (ADR 0005). Nothing a Pass works out is stored in the History |
| `ui/` | BubbleTea layout model orchestrating five components: Toolbar, LogView, Search Bar, Prompt, Help |
| `ui/components/` | `LogView` (viewport + Stream poll), `Toolbar` (Help, Scroll and Search keys), `SearchBar` (one-line textinput under the Log View; owns `enter`, like any text input), `Prompt` (togglable textarea), `Help` (scrollable Keymap table) |
| `config/` | The Keymap: `keymap.json` embedded at build, merged by Action with `~/.config/logCtrl/keymap.json` (see ADR 0003). `LoadKeymap` errors on a bad user file (unknown Action or key name, duplicate or Reserved key, printable `close` or `close_search`); `main` exits on it |
| `ui/utils/` | Responsive sizing system (`Ratio`, `Fixed`, `Modifier` types) and ANSI color constants |
| `utils/` | `SIGWINCH` handler with callback pattern |
| `internal/` | Test-only helpers (`testutil`, `fakestream`) |

Domain terms are defined in `GLOSSARY.md`; design decisions in `docs/adr/`.

### Key Interfaces

**`reader.Stream`** — the bridge from the pipe:
```go
Start(chan bool)    // begin consuming from pipe; notified after each line is held
SetBufferSize(int)  // set the Window height (held lines are kept)
Lines() []Line      // lines in the Window
MoveToEnd()         // Window follows the newest lines
MoveBy(int) int     // scroll the Window; returns lines actually moved
Close()
```

**`cmd.Stream`** — what the UI uses (the `Executor`): a `reader.Stream` whose `Lines()` have every Pass run over them, plus:
```go
GetWindow() string        // text of the lines in the Window after every Pass, for rendering
Search(regex string) error // start a Search (on enter in the Search Bar); "" ends it (on close_search). A bad regex ends it and is returned
```

All UI components implement `tea.Model` (`Init`, `Update`, `View`).

### UI Key Bindings

Keys are dispatched by Action from the merged Keymap (`config/keymap.json`, plus the user's file; ADR 0003): `ui/layout.go` turns a keypress into an Action once, and components get Action-level messages (`components.TeaScroll`), never raw keys. Never match a literal key in the UI; add an Action to the Keymap instead. The built-in keys:

| Key | Action |
|---|---|
| `h` | Open the Help (while prompt and help closed) |
| `Tab` | Open prompt panel (while help closed) |
| `Esc` | Close the prompt or the help (while open); the only Keymap key honoured in the prompt. Cancels typing in the Search Bar: back to the entered regex, or closed if none |
| `q` | Quit (while prompt and help closed) |
| `k` / `j` | Scroll the log view one line up (older) / down (newer); scroll the help while it is open |
| `G` | Follow the newest lines (while prompt and help closed; not shown in the Toolbar) |
| `/` | Open the Search Bar, or edit its regex (while prompt and help closed); `enter` makes it read-only and starts the Search (ignored when empty) |
| `ctrl+l` | Close the Search Bar, ending the Search (while typing in it or while it is read-only) |
| `^C` / `^D` | Reserved, outside the Keymap: always Quit while the prompt is closed (including while typing in the Search Bar) |

### Environment Variables

- `logctrl_child` — integer file descriptor; set by parent on re-exec to signal child mode.
