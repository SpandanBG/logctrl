---
status: accepted
---

# Passes run over the Window on every read, not stored on held Lines

A Search, and the formatting that will follow it, adds context to each Line, such as its Matches. That context depends on the current regex, while the Lines held in the Live Buffer and Chunks never change once built. So an Executor in `cmd/` sits between the reader's Stream and the UI and wraps it. Each time the UI reads the Window, the Executor takes copies of the Window's Lines, sets each copy's shown text to its Text, runs every Pass over them in order, and only then joins their shown text for display. Each Pass formats the shown text itself, with its own style, and the Search always runs last, over the shown text as the other Passes leave it, so its Matches are found in what the user sees. Nothing a Pass works out is written back into the History. This keeps Matches always in step with the regex and with whatever the Window shows, after Scroll, resize and new lines, with no invalidation. The cost is running every Pass over about one screenful of Lines on each render. We rejected matching once on enter (lines that arrive later or come into view by Scroll would go unmatched), and storing Matches on every held Line (more work, and every Line must be cleared when the regex changes).

## Consequences

- `reader.Stream` returns `Lines()`, not text. A Line's Text stays as it came from the Log Feed; only its shown text is formatted, and the Executor's `GetWindow` joins the shown text after every Pass has run.
- The Executor runs every Pass, each doing nothing while idle (the Search while no regex is entered), with the Search held apart so it runs after the others (see ADR 0006), and only the Gutter Pass after it (see ADR 0009).
- A Pass runs over the Window only, so it can't find a Match outside it. Stepping to a Line with a Match outside the Window is done by the Search, which scans the History through a port the Executor gives it (see ADR 0007 and ADR 0008).
- The Search matches the shown text as it stands, so a Log Feed, or a Pass run before the Search, that brings its own ANSI color codes can have a Match split or overlap an escape code and garble its color when highlighted. This is a known limitation, left for a later phase.
- `GetWindow` runs inside a `tea.Cmd`, on a different goroutine from `Update`, where the UI updates the Passes. So the Executor guards its Passes with a mutex.
- Amended (#23): `GetWindow` is gone. The UI reads the Window through `headless.Logs.Window()`, which takes the Executor's `Lines()` and gives their shown text and Gutter cells as slices (see ADR 0010); it still reads them in a `tea.Cmd`, so the mutex stays.
- Amended (#24): `reader.Stream` no longer has `Lines()`. The Window is in `cmd`, held by the Executor and guarded by the same mutex, and reads the History through `Read` (see ADR 0011).
