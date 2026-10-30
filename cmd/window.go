package cmd

import "github.com/SpandanBG/logctrl/reader"

// window - the range of Line Numbers [start, end) being shown, read from the
// History through a reader.Stream. While Following, the range is derived on
// read from the newest Lines and start/end are not used (see docs/adr/0002).
type window struct {
	stream    reader.Stream // the History read from
	height    int           // the Lines shown, as tall as the Log View
	start     int           // Line Number of the first Line shown, while Pinned
	end       int           // Line Number after the last Line shown, while Pinned
	following bool          // shows the newest Lines, whatever start and end are
}

// ------------------------- Private

// newWindow - a Following window over the History of `stream`.
func newWindow(stream reader.Stream) *window {
	return &window{stream: stream, following: true}
}

// lines - the Lines in the window.
func (w *window) lines() []reader.Line {
	for {
		w.clamp()
		from, to := w.start, w.end
		if w.following {
			to = w.stream.Total() // read once, so a Line arriving meanwhile is not taken in
			from = max(w.stream.First(), to-w.height)
		}
		lines := w.stream.Read(from, to)
		if from >= w.stream.First() {
			return lines
		}
		// the History now starts at its oldest held line: clamp again
	}
}

// resize - makes the window `height` Lines tall.
func (w *window) resize(height int) {
	w.height = height
}

// moveBy - Scrolls the window `n` Lines (negative is up / older, positive is
// down / newer), stopping at either edge. Reaching the newest Line makes it
// Following.
func (w *window) moveBy(n int) {
	w.lines() // clamps the window if its Lines can no longer be read
	start := w.lastStart()
	if !w.following {
		start = w.start
	}

	newStart := w.reachable(start + n)
	if newStart == w.lastStart() {
		w.following = true
		return
	}
	w.following = false
	w.start = newStart
	w.end = newStart + w.height
}

// pinCentredOn - Pins the window with line `number` in its middle (the upper
// middle row if its height is even), stopping at the first and newest Lines.
func (w *window) pinCentredOn(number int) {
	w.following = false
	w.start = w.reachable(number - (w.height-1)/2)
	w.end = w.start + w.height
}

// follow - makes the window Following.
func (w *window) follow() {
	w.following = true
}

// reachable - the window start nearest `start` whose Lines can be read.
func (w *window) reachable(start int) int {
	return min(max(start, w.stream.First()), w.lastStart())
}

// clamp - shifts a Pinned window forward to the first Line that can be read
// (see reader.Stream's First). If that brings it to the newest Line it Follows
// again.
func (w *window) clamp() {
	first := w.stream.First()
	if w.following || w.start >= first {
		return
	}

	w.end += first - w.start
	w.start = first
	if w.end >= w.stream.Total() {
		w.following = true
	}
}

// lastStart - start of the window when it shows the newest Lines.
func (w *window) lastStart() int {
	return max(w.stream.First(), w.stream.Total()-w.height)
}
