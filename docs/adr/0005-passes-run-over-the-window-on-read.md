---
status: accepted
---

# Passes run over the Window on every read, not stored on held Lines

A Search, and the formatting that will follow it, adds context to each Line, such as its Matches. That context depends on the current regex, while the Lines held in the Live Buffer and Chunks never change once built. So an Executor in `cmd/` sits between the reader's Stream and the UI and wraps it. Each time the UI reads the Window, the Executor takes copies of the Window's Lines, runs every Pass over them in order, and only then joins their text for display. Nothing a Pass works out is written back into the History. This keeps Matches always in step with the regex and with whatever the Window shows, after Scroll, resize and new lines, with no invalidation. The cost is running every Pass over about one screenful of Lines on each render. We rejected matching once on enter (lines that arrive later or come into view by Scroll would go unmatched), and storing Matches on every held Line (more work, and every Line must be cleared when the regex changes).

## Consequences

- `reader.Stream` returns `Lines()`, not text. Only the Executor's `GetWindow` turns Lines into text, after every Pass has run.
- A Pass only ever sees the Window, so it can't find a Match outside it. Moving the Window to an older Match will need its own way of finding one, such as scanning the History or the Log File.
- `GetWindow` runs inside a `tea.Cmd`, on a different goroutine from `Update`, where `Search` is called. So the Executor guards its Passes with a mutex.
