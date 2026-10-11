# logctrl

A terminal log viewer: it consumes a stream of log lines from another process and lets the user watch them live or scroll back through recent history.

## Ingest

**Log Feed**:
The incoming stream of log lines produced by the process being watched.
_Avoid_: input, pipe, source

**Log File**:
The complete on-disk copy of everything received on the Log Feed during a session.
_Avoid_: temp file, log dump

**Line Number**:
A line's position in the Log Feed since the session started; it never changes, even after the line leaves the Live Buffer.
_Avoid_: index, offset, position

**Total**:
The number of lines received on the Log Feed so far.
_Avoid_: count, length

## Display

**Live Buffer**:
The most recent lines held in memory, up to its Capacity; the only lines the user can see.
_Avoid_: LAB, live access buffer, scrollback, history

**Capacity**:
The maximum number of lines the Live Buffer holds before the oldest are evicted.
_Avoid_: buffer size, scrollback size

**Oldest Held**:
The Line Number of the oldest line still in the Live Buffer.

**Window**:
The contiguous range of Line Numbers currently shown to the user, as tall as the log view.
_Avoid_: RAB, random access buffer, viewport, page

**Following**:
The Window mode in which it always shows the newest lines.
_Avoid_: live mode, tailing, auto-scroll

**Pinned**:
The Window mode in which it keeps showing the same Line Numbers while new lines arrive.
_Avoid_: frozen, paused, locked

**Clamp**:
Shifting a Pinned Window forward when its first line is no longer held in the Live Buffer.
