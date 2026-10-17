package reader

import (
	"io"
	"strings"
)

// History - every line of the session by line number: the newest from the
// LiveBuffer, older ones from Chunks of `chunkSize` lines loaded from the Log
// File through a sparse line index (see docs/adr/0004).
type History struct {
	lb        *LiveBuffer
	logFile   io.ReaderAt
	chunkSize int
	index     []int64          // index[c] - byte offset in the Log File of line c*chunkSize
	size      int64            // bytes of the Log File taken up by pushed lines
	chunks    map[int][]string // at most `maxChunks`, by chunk number
	unread    bool             // the Log File failed to read; only the LiveBuffer is shown
}

// maxChunks - Chunks held in memory: the one being viewed and its neighbours.
const maxChunks = 3

// NewHistory - Creates a history holding the newest `capacity` lines in memory
// and reading older ones from `logFile` in Chunks of `chunkSize` lines.
func NewHistory(capacity, chunkSize int, logFile io.ReaderAt) *History {
	return &History{
		lb:        NewLiveBuffer(capacity),
		logFile:   logFile,
		chunkSize: chunkSize,
		chunks:    map[int][]string{},
	}
}

// Push - appends a line as written to the Log File, terminator included.
func (h *History) Push(raw string) {
	if h.lb.Total()%h.chunkSize == 0 {
		h.index = append(h.index, h.size)
	}
	h.size += int64(len(raw))
	h.lb.Push(text(raw))
}

// Total - number of lines received so far.
func (h *History) Total() int {
	return h.lb.Total()
}

// First - line number of the oldest line that can be shown: the first line of
// the session, or the oldest held once the Log File has failed to read.
func (h *History) First() int {
	if h.unread {
		return h.lb.OldestHeld()
	}
	return 0
}

// Lines - returns the lines with line numbers in [from, to). Callers must keep
// the range within [First(), Total()]. `false` if the Log File failed to read,
// after which First() has moved up to the oldest held line.
func (h *History) Lines(from, to int) ([]string, bool) {
	oldest := h.lb.OldestHeld()
	lines := make([]string, 0, to-from)
	for n := from; n < min(to, oldest); n += 1 {
		chunk, ok := h.chunk(n / h.chunkSize)
		if !ok {
			return nil, false
		}
		lines = append(lines, chunk[n%h.chunkSize])
	}
	if to > oldest {
		lines = append(lines, h.lb.Lines(max(from, oldest), to)...)
	}
	return lines, true
}

// ----------------------- PRIVATE

// chunk - the lines of Chunk `c`, loaded from the Log File if not held.
// `false` if the Log File failed to read.
func (h *History) chunk(c int) ([]string, bool) {
	if lines, ok := h.chunks[c]; ok {
		return lines, true
	}

	end := h.size
	if c+1 < len(h.index) {
		end = h.index[c+1]
	}
	buf := make([]byte, end-h.index[c])
	if _, err := h.logFile.ReadAt(buf, h.index[c]); err != nil {
		h.unread = true
		return nil, false
	}

	lines := strings.SplitAfter(string(buf), "\n")
	for i, raw := range lines {
		lines[i] = text(raw)
	}
	if len(h.chunks) == maxChunks {
		h.dropFurthest(c)
	}
	h.chunks[c] = lines
	return lines, true
}

// dropFurthest - drops the held Chunk furthest from Chunk `c`.
func (h *History) dropFurthest(c int) {
	furthest := c
	for held := range h.chunks {
		if abs(held-c) > abs(furthest-c) {
			furthest = held
		}
	}
	delete(h.chunks, furthest)
}

// text - a raw line without its terminator.
func text(raw string) string {
	return strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
}

func abs(n int) int {
	return max(n, -n)
}
