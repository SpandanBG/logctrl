# Architecture

`logctrl` is a terminal UI log viewer that reads log streams from stdin and displays them interactively. It uses a **parent-child process model**:

- **Parent** (`main.go`): Opens `/dev/tty` in raw mode, creates a pipe, re-execs itself as a child with `logctrl_child=<fd>` env var, then multiplexes I/O between PTY master/slave and stdin. Handles `SIGWINCH` (resize) and `SIGTERM` (kills process group on exit).
- **Child** (`main.go → startChildProcess`): Detects `logctrl_child` env var, reads logs from that file descriptor via `Stream`, wraps it in the `cmd.Executor` with every Pass (built idle; the Search, then the Gutter Pass, last), and launches the BubbleTea UI; the Passes and the UI get lipgloss's default renderer, passed in so tests can force colors.

## Data Flow

```
stdin (log producer) ──→ anonymous pipe ──→ Stream ──→ History (LiveBuffer + Chunks) ──→ Window ──→ Executor (Passes, Search, Gutter) ──→ LogView (Gutter + viewport)
                                       └──→ temp file (logCtrl_logFile.txt in os.TempDir()) ──→ Chunks

/dev/tty ──→ PTY slave (child) ──→ BubbleTea event loop
PTY master ──→ stdout (rendered UI)
```

## Key Packages

| Package | Role |
|---|---|
| `reader/` | `Stream` tees the Log Feed into the Log File and a `History`: a `LiveBuffer` (bounded ring addressed by Line Number) plus older lines loaded from the Log File in Chunks via a sparse Line Index (ADR 0004). A `Window` (two Line Numbers + Following/Pinned) selects what is shown, back to Line Number 0. Each held line is a `Line` (its Line Number and text; its shown text and Matches once the Passes have run), built once on push or Chunk load; `Window.Lines` and `Stream.Lines` return `[]Line` |
| `cmd/` | The `Executor` wraps a `reader.Stream` and is the `cmd.Stream` the UI uses: on each read it sets the shown text of copies of the Window's Lines to their text, runs every Pass over them, then the Search (`search.go`: marks each Line's Matches for the Search regex and highlights them black on yellow, and sets `GutterContext.Search.Current` on the Current Line), then the Gutter Pass (`gutter.go`: turns each Line's `GutterContext` into its two `Gutter` cells, `>` in cell 1 beside the Current Line), then joins their shown text and their Gutter cells (ADR 0005, ADR 0009). Each Pass builds its own lipgloss style from the renderer it is built with, and has the same `Update(msg any) error`: the UI hands a typed message (`SearchFor{Regex}`) to the Executor's `Update`, which hands it to every Pass; each acts on its own messages and ignores the rest (ADR 0006). The Executor only runs the Passes together (hands each message to every Pass, runs every Pass, joins the shown text and Gutter cells); all Search logic is in `search.go`. On `PreviousMatch{}` and `NextMatch{}` (broadcast like any other; the first step finds the Current Line) the Search finds and steps between Lines with a Match through a narrow port the Executor attaches (`history`: Window, Total, First, Read as the Passes before the Search leave the Lines with the lock released, PinCentredOn), scanning the History a batch at a time, Pins the Window centred on the Current Line (marked in the Gutter), cancels its scan on any other message, and remembers the Lines with a Match it has found until then; `Follow{}` drops the Current Line (ADR 0007, ADR 0008). Nothing a Pass works out is stored in the History |
| `ui/` | BubbleTea layout model orchestrating six components: Toolbar, LogView, Filter Bar, Search Bar, Prompt, Help |
| `ui/components/` | `LogView` (viewport + Stream poll, with a `gutterView` inside its border on the left: the Gutter cells from the same `GetWindow` read, on background 236, as tall as the Log View), `Toolbar` (Help, Scroll, Search, Filter and Match keys), `RegexBar` (textinput under the Log View with a grey line along its top, built as the Search Bar or the Filter Bar (above the Search Bar), whose prompt `/:` or `f:` is black on green or purple in every state, then a space; owns its lifecycle (hidden, typing, read-only: shown on edit, hidden on close, or on cancel with no regex entered) and tells the layout through `Shown`, `Typing`, `Entered` and a fixed `Height` (0 or 2); owns `enter`, like any text input, and says which bar the regex was entered in; dims the regex while read-only, and shows a bad regex's error in red after the prompt, cut to the bar's width with `…`), `Prompt` (togglable textarea), `Help` (scrollable Keymap table) |
| `config/` | The Keymap: `keymap.json` embedded at build, merged by Action with `~/.config/logCtrl/keymap.json` (see ADR 0003). `LoadKeymap` errors on a bad user file (unknown Action or key name, duplicate or Reserved key, printable `close`, `close_search` or `close_filter`); `main` exits on it |
| `ui/utils/` | Responsive sizing system (`Ratio`, `Fixed`, `Modifier` types) and ANSI color constants |
| `utils/` | `SIGWINCH` handler with callback pattern |
| `internal/` | Test-only helpers (`testutil`, `fakestream`) |

Domain terms are defined in `GLOSSARY.md`; design decisions in `docs/adr/`.

## Key Interfaces

**`reader.Stream`** — the bridge from the pipe:
```go
Start(chan bool)          // begin consuming from pipe; notified after each line is held
SetBufferSize(int)        // set the Window height (held lines are kept)
Lines() []Line            // lines in the Window
Total() int               // lines received so far
First() int               // Line Number of the oldest line that can be read (moves up if the Log File fails to read)
Read(from, to int) []Line // lines with Line Numbers in [from, to), wherever the Window is
MoveToEnd()               // Window follows the newest lines
MoveBy(int) int           // scroll the Window; returns lines actually moved
PinCentredOn(int)         // Pin the Window with that Line Number in its middle, stopping at either end
Close()
```

**`cmd.Stream`** — what the UI uses (the `Executor`): a `reader.Stream` whose `Lines()` have every Pass run over them, plus:
```go
GetWindow() (shown, gutter string) // shown text (Matches highlighted) and Gutter cells of the lines in the Window after every Pass, each joined by newlines, from one read, for rendering
Update(msg any) error              // hand msg to every Pass, errors joined: SearchFor{Regex} starts a Search (on enter in the Search Bar), SearchFor{} ends it (on close_search). A bad regex ends it and is returned; a new regex makes the Window Following, the same one again is not sent. PreviousMatch{} and NextMatch{} (up/down, in a tea.Cmd) find the Current Line on the first step, then step between Lines with a Match. Follow{} (on the follow key, before the Window Follows) drops the Current Line
```

All UI components implement `tea.Model` (`Init`, `Update`, `View`).

## UI Key Bindings

Keys are dispatched by Action from the merged Keymap (`config/keymap.json`, plus the user's file; ADR 0003): `ui/layout.go` turns a keypress into an Action once, and components get Action-level messages (`components.TeaScroll`), never raw keys. Never match a literal key in the UI; add an Action to the Keymap instead. The built-in keys:

| Key | Action |
|---|---|
| `h` | Open the Help (while prompt and help closed) |
| `Tab` | Open prompt panel (while help closed) |
| `Esc` | Close the prompt or the help (while open); the only Keymap key honoured in the prompt. Cancels typing in the Search Bar: back to the entered regex (or its error), or closed if none |
| `q` | Quit (while prompt and help closed) |
| `k` / `j` | Scroll the log view one line up (older) / down (newer); scroll the help while it is open |
| `G` | Follow the newest lines (while prompt and help closed; not shown in the Toolbar); drops the Current Line, keeping the Search |
| `/` | Open the Search Bar, or edit its regex (while prompt and help closed); `enter` makes it read-only and starts the Search, highlighting every Match and making the Window Following (ignored when empty; changes nothing for the regex already entered); a bad regex shows its error in red, and `/` brings the regex back to fix |
| `ctrl+l` | Close the Search Bar, ending the Search (while typing in it or while it is read-only); once a regex has been entered, also Follows the newest lines |
| `f` | Open the Filter Bar above the Search Bar, or edit its regex (while prompt and help closed); `enter` makes it read-only (ignored when empty), `esc` cancels typing like the Search Bar. Phase 1: the regex is not yet sent to the Executor, so nothing is hidden |
| `ctrl+f` | Close the Filter Bar, clearing it (while typing in it or while it is read-only) |
| `up` / `down` | Step to the previous (older) / next (newer) Line with a Match, centring the Window on it (while a regex is entered and the Search Bar is read-only, with the prompt and help closed); with no Current Line, either finds it: the newest Line with a Match at or above the Window's bottom, Pinning the Window (nothing if no Line matches) |
| `^C` / `^D` | Reserved, outside the Keymap: always Quit while the prompt is closed (including while typing in the Search Bar or the Filter Bar) |

While typing in the Search Bar or the Filter Bar, only `esc`, that bar's own close key and `^C` / `^D` act; every other key, the other bar's keys included, is typed.

## Environment Variables

- `logctrl_child` — integer file descriptor; set by parent on re-exec to signal child mode.
