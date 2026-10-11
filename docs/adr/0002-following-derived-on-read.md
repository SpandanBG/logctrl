---
status: accepted
---

# Following is derived on read, not driven by a tracker goroutine

The original requirement was a goroutine that slides the Window forward as lines arrive. Instead, Following is an explicit flag and the Window's range is computed from Total and the Window height whenever it is read; nothing moves on ingest. A tracker goroutine would add a channel, a shutdown path, and a gap where Total has advanced but the Window hasn't — during which "end == Total" misreports the mode. Do not reintroduce one to own Window state; a future goroutine (e.g. throttled re-render) should consume the `next` notification instead.

## Considered Options

- Dedicated tracker goroutine — rejected for the staleness gap and extra lifecycle.
- Advance the Window inside the existing ingest goroutine — correct, but does work on every line even when nobody is reading.
