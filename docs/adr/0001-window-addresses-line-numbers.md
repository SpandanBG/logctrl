---
status: accepted
---

# Window addresses lines by Line Number, not by Live Buffer slot

The Live Buffer is a ring with a fixed Capacity (independent of screen height), so a line's slot in it changes as the ring wraps. The Window therefore stores Line Numbers (`start`, `end` as `int`) instead of slot indices, so a Pinned Window keeps showing the same lines while new ones arrive; when its lines are evicted it is Clamped forward to Oldest Held, and if that brings its end to Total it becomes Following. We rejected keeping the Live Buffer at screen height (nothing to scroll) and backing the Window with the on-disk Log File (unbounded index, file I/O on every scroll).

## Consequences

- History older than Capacity is only in the Log File and cannot be scrolled to.
- Capacity is a constant in `reader/stream.go` for now; a `-capacity N` flag (fatal on invalid) is deferred — note the child is re-exec'd without args, so a flag must be forwarded explicitly.
- Window behaviour on terminal resize while Pinned is deliberately undefined (out of scope).
