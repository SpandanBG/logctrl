# Architecture

`logctrl` is a terminal UI log viewer that reads log streams from stdin and displays them interactively. It uses a **parent-child process model**:

- **Parent** (`main.go`): Opens `/dev/tty` in raw mode, creates a pipe, re-execs itself as a child with `logctrl_child=<fd>` env var, then multiplexes I/O between PTY master/slave and stdin. Handles `SIGWINCH` (resize) and `SIGTERM` (kills process group on exit).
- **Child** (`main.go → startChildProcess`): Detects `logctrl_child` env var, reads logs from that file descriptor via `Stream`, wraps it in the `cmd.Executor`, which builds its own Passes idle (the Filter, first, then the Search, then the Gutter Pass, last), hands that to `headless.New`, and launches the BubbleTea UI over the `headless.Logs`; the Executor (for the Search) and the UI get lipgloss's default renderer, passed in so tests can force colors.

## Data Flow

```
stdin (log producer) ──→ anonymous pipe ──→ Stream ──→ History (LiveBuffer + Chunks) ──→ Executor (Window, Passes, Search, Gutter) ──→ headless.Logs (Window, notices) ──→ LogView (Gutter + viewport)
                                       └──→ temp file (logCtrl_logFile.txt in os.TempDir()) ──→ Chunks

/dev/tty ──→ PTY slave (child) ──→ BubbleTea event loop
PTY master ──→ stdout (rendered UI)
```

## Key Packages

| Package | Role |
|---|---|
| `reader/` | `Stream` tees the Log Feed into the Log File and a `History`: a `LiveBuffer` (bounded ring addressed by Line Number) plus older lines loaded from the Log File in Chunks via a sparse Line Index (ADR 0004). `Stream.Read` gives the Lines in a range of Line Numbers, back to Line Number 0. Each held line is a `Line` (its Line Number and text; its shown text, Matches and `Filter.Hide` once the Passes have run), built once on push or Chunk load |
| `cmd/` | The `Executor`: the Window over a `reader.Stream`, with the Passes run over it on each read, as the `cmd.Stream` that `headless` uses. See [`cmd/`](#cmd) below |
| `headless/` | `Logs`, the Log Feed as any UI sees it (ADR 0010), and `GutterWidth`, so the UI never imports `reader`: `New(cmd.Stream)` starts the stream, its Window one Line tall until the first `Resize`; `Window()` gives the shown text and the Gutter cells (an empty cell a space) of each Line the Filter does not hide, as two slices from one read; `Update(msg)` hands every message to the Executor; `Changed()` holds at most one notice, left without blocking after each new Line and after every `Update`, so notices not yet taken collapse and the reader never waits for the UI, and closes once the stream is; `Close()` closes the stream once, however often it is called |
| `ui/` | BubbleTea layout model orchestrating six components: Toolbar, LogView, Filter Bar, Search Bar, Prompt, Help |
| `ui/components/` | `LogView` (viewport drawn on each `headless.Logs` notice, sending `Resize` with its inner height, with a `gutterView` inside its border on the left: the Gutter cells from the same `Window()` read, on background 236, as tall as the Log View; the layout sends `Scroll` and `Follow` to the `headless.Logs` itself), `Toolbar` (Help, Scroll, Search, Filter and Match keys), `RegexBar` (textinput under the Log View with a grey line along its top, built as the Search Bar or the Filter Bar (above the Search Bar), whose prompt `/:` or `f:` is black on green or purple in every state, then a space; owns its lifecycle (hidden, typing, read-only: shown on edit, hidden on close, or on cancel with no regex entered) and tells the layout through `Shown`, `Typing`, `Entered` and a fixed `Height` (0 or 2); owns `enter`, like any text input, and says which bar the regex was entered in; dims the regex while read-only, and shows a bad regex's error in red after the prompt, cut to the bar's width with `…`), `Prompt` (togglable textarea), `Help` (scrollable Keymap table) |
| `config/` | The Keymap: `keymap.json` embedded at build, merged by Action with `~/.config/logCtrl/keymap.json` (see ADR 0003). `LoadKeymap` errors on a bad user file (unknown Action or key name, duplicate or Reserved key, printable `close`, `close_search` or `close_filter`); `main` exits on it |
| `ui/utils/` | Responsive sizing system (`Ratio`, `Fixed`, `Modifier` types) and ANSI color constants |
| `utils/` | `SIGWINCH` handler with callback pattern |
| `internal/` | Test-only helpers (`testutil`, `logfeed`, `logview`) |

Domain terms are defined in `GLOSSARY.md`; design decisions in `docs/adr/`.

### `cmd/`

- **Executor** (`executor.go`): holds a `reader.Stream` (the History) and the Window, and guards both the Window and the Passes with one lock. It builds the Filter, the Search and the Gutter Pass itself; the Search builds its highlight style from the renderer passed to `NewExecutor(stream, renderer, others...)`. Apart from the Window, it only runs the Passes together: it hands each message to every Pass and runs every Pass. All Search logic is in `search.go`.
- **Window** (`window.go`): two Line Numbers plus Following/Pinned and a height, read from the History through `Read`, `Total` and `First`. It is Clamped forward to `First` once Lines can no longer be read (ADR 0001, ADR 0011). It is given the Passes before the Search, so it can take in Lines while the Filter hides some (see **Take-in**).
- **Passes, on each read:** the Executor sets the shown text of copies of the Window's Lines to their text, then runs its Passes over them in a fixed order (ADR 0005, ADR 0009):
  1. the Filter (`filter.go`): sets `Filter.Hide` on each Line whose shown text, still its text, its regex doesn't match anywhere, empty matches included;
  2. any other Passes given to `NewExecutor`;
  3. the Search (`search.go`): marks each Line's Matches for the Search regex, highlights them black on yellow, and sets `GutterContext.Search.Current` on the Current Line;
  4. the Gutter Pass (`gutter.go`): turns each Line's `GutterContext` into its two `Gutter` cells, `>` in cell 1 beside the Current Line.

  Nothing a Pass works out is stored in the History.
- **Take-in:** while the Filter hides Lines, the Window takes in older Lines through `Read` (newer ones once it reaches the first Line), running the Passes before the Search over them, until it has as many shown Lines as the Window's height. It reads at most `reader.DefaultCapacity` Lines beyond the Window and stops once a read finds nothing. The Window itself is unchanged, so ending the Filter gives it back as it was (ADR 0009, Amended (#16) and (#26)). The Search's port still sees the Window without the Lines taken in (#18).
- **Messages:** every Pass has the same `Update(msg any) error`, and acts on its own messages and ignores the rest (ADR 0006). The UI hands a typed message (`SearchFor{Regex}`, `FilterFor{Regex}`, and the Window's own `Scroll{N}`, `Resize{Height}` and `Follow{}`, all defined in `cmd`) through `headless` to the Executor's `Update`. That hands it to every Pass but `Resize`, then moves the Window on `Scroll`, `Resize` and `Follow`, under its lock (ADR 0011).
- **Stepping:** `PreviousMatch{}` and `NextMatch{}` are broadcast like any other message; the first step finds the Current Line (ADR 0007, ADR 0008). The Search steps between Lines with a Match through a narrow port the Executor attaches (`history`: Window, Total, First, Read as the Passes before the Search leave the Lines, with the lock released, PinCentredOn, and Follow). It:
  - scans the History a batch at a time, passing over Lines the Filter hides;
  - Pins the Window centred on the Current Line, which is marked in the Gutter;
  - cancels its scan on any other message, a Scroll included;
  - remembers the Lines with a Match it has found until then.

  `Follow{}` drops the Current Line.

## Key Interfaces

**`reader.Stream`** — the bridge from the pipe, the History of the Log Feed by Line Number:
```go
Start(chan bool)          // begin consuming from pipe; notified after each line is held
Total() int               // lines received so far
First() int               // Line Number of the oldest line that can be read (moves up once the Log File fails to read)
Read(from, to int) []Line // lines with Line Numbers in [from, to), leaving out any that can no longer be read
Close()
```

**`cmd.Stream`** — what `headless` uses (the `Executor`), the Window over a `reader.Stream`:
```go
Start(next chan bool)              // start reading the Log Feed, notifying next as each Line arrives
Lines() []reader.Line              // the Lines in the Window after every Pass, grown to the Window's height while the Filter hides Lines
Close()                            // stop reading the Log Feed
Update(msg any) error              // Scroll{N} is handed to every Pass, then Scrolls the Window (up to the first Line, down to the newest, which Follows); Resize{Height} only makes the Window that tall, reaching no Pass; Follow{} is handed to every Pass, then the Window Follows. Every other msg goes to every Pass, errors joined: SearchFor{Regex} starts a Search (on enter in the Search Bar), SearchFor{} ends it (on close_search). A bad regex ends it and is returned; a new regex, a bad one included, makes the Window Following (the Search does, through its port), and the same one again is ignored by the Search, keeping the Current Line. PreviousMatch{} and NextMatch{} (up/down, in a tea.Cmd) find the Current Line on the first step, then step between Lines with a Match. Follow{} (on the follow key, before the Window Follows) drops the Current Line. FilterFor{Regex} starts a Filter (on enter in the Filter Bar), FilterFor{} ends it (on close_filter); a bad regex ends it and is returned, and the Window keeps its mode. The same one again hides the same Lines, but reaches every Pass like any other message, so the Search cancels a running step and forgets the Lines with a Match it has found (#28). A Current Line the Filter hides stays the Current Line
```

**`headless.Logs`** — what the UI uses:
```go
Window() (shown, gutter []string) // shown text (Matches highlighted) and Gutter cells of each Line in the Window the Filter does not hide, after every Pass, from one read
Changed() <-chan struct{}         // notified after each new Line and after every Update; notices not yet taken collapse into one
Update(msg any) error             // hands msg to the Executor's Update, as in cmd.Stream
Close()                           // closes the stream once; Changed closes soon after
```

All UI components implement `tea.Model` (`Init`, `Update`, `View`).

## UI Key Bindings

Keys are dispatched by Action from the merged Keymap (`config/keymap.json`, plus the user's file; ADR 0003): `ui/layout.go` turns a keypress into an Action once, and components get Action-level messages (`components.TeaHelpScroll`), never raw keys. Never match a literal key in the UI; add an Action to the Keymap instead. The built-in keys:

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
| `f` | Open the Filter Bar above the Search Bar, or edit its regex (while prompt and help closed); `enter` makes it read-only and starts the Filter, hiding every Line its regex doesn't match (ignored when empty; changes nothing for the regex already entered); a bad regex shows its error in red; `esc` cancels typing like the Search Bar. Hidden Lines are left out and the Window takes in others to fill the Log View |
| `ctrl+f` | Close the Filter Bar, ending the Filter (while typing in it or while it is read-only); the Window keeps its mode |
| `up` / `down` | Step to the previous (older) / next (newer) Line with a Match, centring the Window on it (while a regex is entered and the Search Bar is read-only, with the prompt and help closed); with no Current Line, either finds it: the newest Line with a Match at or above the Window's bottom, Pinning the Window (nothing if no Line matches) |
| `^C` / `^D` | Reserved, outside the Keymap: always Quit while the prompt is closed (including while typing in the Search Bar or the Filter Bar) |

While typing in the Search Bar or the Filter Bar, only `esc`, that bar's own close key and `^C` / `^D` act; every other key, the other bar's keys included, is typed.

## Environment Variables

- `logctrl_child` — integer file descriptor; set by parent on re-exec to signal child mode.
