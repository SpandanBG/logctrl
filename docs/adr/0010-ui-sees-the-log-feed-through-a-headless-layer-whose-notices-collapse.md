---
status: accepted
---

# The UI sees the Log Feed through a headless layer whose notices collapse

The Log View called the Executor directly: it started the stream, set its height, moved the Window, joined the shown text and Gutter cells it was given, and waited on the stream's channel for each new Line. Every redraw that was not a new Line (a Scroll, a Follow, a step, a Search or Filter starting or ending) had to be asked for by the UI itself, and the reader waited for the UI to take each Line before reading the next. So any UI built in the app would have to repeat all of that. A headless layer, `headless.Logs`, now sits between the Executor and the UI, and the UI uses nothing else of the Log Feed: `Window()` gives the shown text and the Gutter cells of each Line the Filter does not hide, as two slices from one read; `Update(msg)` takes every message, the Passes' (`SearchFor`, `FilterFor`, `PreviousMatch`, `NextMatch`, `Follow`) and the Window's (`Scroll`, `Resize`), all defined in `cmd`, and hands all but `Resize` on to the Passes; `Changed()` notifies the UI when the Window may show something different; `Close()` ends it, and `Changed()` then closes. A notice is sent after each new Line and after every `Update`, whatever the message, and holds at most one pending notice: sending never blocks, so changes the UI has not yet drawn collapse into one notice and the reader no longer waits for the UI. We rejected growing the Executor into this interface (it would mix running the Passes with starting, sizing, moving and notifying), notifying only on a real change (`headless` would have to work out whether the Window differs, while a spurious redraw is harmless), and keeping the reader paced by the UI (a slow render would hold up the Log Feed).

## Consequences

- The UI imports `headless` and `cmd`'s message types, never `reader`. `main` builds the Executor and hands it to `headless.New`, which starts it at height 1 until the first `Resize`.
- `cmd.Stream` is a `reader.Stream` plus `Update`; `GetWindow` is gone, and turning Lines into rows lives in `headless`.
- The Log View only draws: it sends `Resize` with its inner height and draws on each notice. The layout sends `Scroll` and `Follow`, and no longer asks for redraws after a step or a Search or Filter change.
- `Follow{}` is handed to the Executor, dropping the Current Line, before the Window is made Following, so no step scan still running can Pin the Window after it.
- `Scroll` reaches the Passes like any other message, so it cancels a step's scan still running and makes the Search forget the Lines with a Match it has found, as any message but a step does (see ADR 0007). `Resize` does not: the Log View sends it whenever a bar or the Prompt opens or closes, or the terminal resizes, none of which should cancel a step.
- The UI may skip drawing some states of the Window between notices; it always draws the latest.
- Amended (#24): the Window lives in the Executor (see ADR 0011), so `headless.Logs.Update` hands every message on, `Resize` included. The Executor moves the Window on `Scroll`, `Resize` and `Follow`, and keeps `Resize` from the Passes itself. `cmd.Stream` is `Start`, `Lines`, `Update` and `Close`.
- Amended (#30): the Executor no longer keeps `Resize` from the Passes. It hands every message to the Window, then every Pass, and the Search ignores `Resize`, so it still cancels no step (see ADR 0011).
