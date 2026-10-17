---
status: accepted
---

# Window reads history beyond the Live Buffer from the Log File in Chunks

Lines older than Oldest Held are only in the Log File, and ADR 0001 left them unreachable because backing the Window with the file meant an unbounded index and file I/O on every scroll. Instead, like a game streaming map chunks around the player, the Window reads through one source that answers from the Live Buffer when it holds the lines and otherwise from Chunks of K lines (1000) loaded from the Log File. A sparse Line Index, the byte offset of every Kth line recorded during ingest, makes a Chunk load one `ReadAt` on the handle the Log Feed is already teed into. At most 3 Chunks are held, and the one furthest from the Window is dropped first. This means moving back and forth across a Chunk boundary never reloads, and a Window can span two Chunks, or a Chunk and the Live Buffer. Chunks load synchronously when the Window needs a line that isn't held, which costs about a millisecond. Lines never change and Oldest Held only moves forward, so a loaded Chunk is never stale. We rejected a full per-line index (unbounded memory), scanning from the top of the file on each load (cost grows with the file), and background prefetch (it adds a loading state we don't need yet).

## Consequences

- Scroll reaches Line Number 0. A Pinned Window keeps its Line Numbers after they are evicted, so Clamp is retired.
- If reading the Log File fails, the source behaves as if nothing older than Oldest Held exists: Scroll stops there, and a Pinned Window moves forward to Oldest Held, which is the old Clamp kept only as a fallback. There is no error UI.
- The Line Index has to count raw bytes per line, including the terminator, because `bufio.Scanner` strips `\n` and `\r`.
- The Window doesn't know about Chunks; it only sees the lines in [from, to) and the bounds 0 and Total.
- The UI doesn't show whether the Window is reading from the Live Buffer or from a Chunk.
- Prefetching neighbouring Chunks is deferred.
