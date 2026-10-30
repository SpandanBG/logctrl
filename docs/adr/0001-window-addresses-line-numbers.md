---
status: accepted
---

# Window addresses lines by Line Number, not by Live Buffer slot

The Live Buffer is a ring with a fixed Capacity (independent of screen height), so a line's slot in it changes as the ring wraps. The Window therefore stores Line Numbers (`start`, `end` as `int`) instead of slot indices, so a Pinned Window keeps showing the same lines while new ones arrive; when its lines are evicted it is Clamped forward to Oldest Held, and if that brings its end to Total it becomes Following. We rejected keeping the Live Buffer at screen height (nothing to scroll) and backing the Window with the on-disk Log File (unbounded index, file I/O on every scroll).

## Consequences

- ~~History older than Capacity is only in the Log File and cannot be scrolled to.~~ Superseded by ADR 0004: history is loaded from the Log File in Chunks, and Clamp is retired except as a read-error fallback.
- Capacity is a constant in `reader/stream.go` for now; a `-capacity N` flag (fatal on invalid) is deferred — note the child is re-exec'd without args, so a flag must be forwarded explicitly.
- Window behaviour on terminal resize while Pinned is deliberately undefined (out of scope).
- Amended (#24): the Window lives in `cmd`, held by the Executor, and reads the History through `reader.Stream`; it still addresses Lines by Line Number and is Clamped forward the same way (see ADR 0011). GLOSSARY's Pinned now reads "its top Line stays the same while new Lines arrive", not "keeps showing the same Line Numbers": once #18 fills a Pinned Window with newer Lines from its top, a Line that arrives can show at the bottom of one that isn't full.
- Amended (#18): a Pinned Window holds one Line Number, its anchor, in place of `start` and `end`. It shows from the first Line the Filter does not hide at or after it, taking in newer Lines, then older ones only once it reaches the newest. Scroll and centring count the Lines the Filter does not hide, looking at most `reader.DefaultCapacity` Lines away (more for a longer Scroll). Clamped forward, it becomes Following once its bottom is the newest Line the Filter does not hide, as a Scroll down does.
