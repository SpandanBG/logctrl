package cmd

import "github.com/SpandanBG/logctrl/reader"

// maxTakenIn - the most Lines beyond the Window a read takes in to fill it
// while the Filter hides Lines: as many as the Live Buffer holds, so a Filter
// that rarely matches never reads the whole History on every render.
const maxTakenIn = reader.DefaultCapacity

// window - the range of Line Numbers [start, end) being shown, read from the
// History through a reader.Stream. While Following, the range is derived on
// read from the newest Lines and start/end are not used (see docs/adr/0002).
type window struct {
	stream    reader.Stream                     // the History read from
	run       func([]reader.Line) []reader.Line // the Passes before the Search, run over the Lines it takes in; called with the Executor's lock held
	height    int                               // the Lines shown, as tall as the Log View
	start     int                               // Line Number of the first Line shown, while Pinned
	end       int                               // Line Number after the last Line shown, while Pinned
	following bool                              // shows the newest Lines, whatever start and end are
}

// ------------------------- Private

// newWindow - a Following window over the History of `stream`, taking in
// Lines while `run`, the Passes before the Search, hides some.
func newWindow(stream reader.Stream, run func([]reader.Line) []reader.Line) *window {
	return &window{stream: stream, run: run, following: true}
}

// shown - the Lines in the window after `run`, with as many Lines taken in
// as it takes to show as many as its height. Callers hold the Executor's lock.
func (w *window) shown() []reader.Line {
	return w.takeIn(w.run(w.lines()))
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

// takeIn - `lines`, with as many older Lines before them, then newer ones
// after them, run over by `run`, as it takes to show as many Lines as the
// window's height; it stops at the first and the newest Line, once a read
// finds nothing, and after maxTakenIn Lines. Callers hold the Executor's lock.
func (w *window) takeIn(lines []reader.Line) []reader.Line {
	first, total, height := w.stream.First(), w.stream.Total(), w.height
	budget := maxTakenIn
	for _, older := range []bool{true, false} {
		for {
			missing := min(height-shownCount(lines), budget) // reads no more Lines than it may show
			if missing <= 0 || len(lines) == 0 {
				return lines
			}
			from, to := lines[0].Number-missing, lines[0].Number
			if !older {
				from = lines[len(lines)-1].Number + 1
				to = from + missing
			}
			from, to = max(from, first), min(to, total)
			if from >= to { // at the first, or the newest, Line
				break
			}
			read := w.run(w.stream.Read(from, to))
			if len(read) == 0 { // the Log File has failed to read
				break
			}
			budget -= len(read)
			if older {
				lines = append(read, lines...)
			} else {
				lines = append(lines, read...)
			}
		}
	}
	return lines
}

// shownCount - how many of `lines` the Filter does not hide.
func shownCount(lines []reader.Line) int {
	count := 0
	for _, line := range lines {
		if !line.Filter.Hide {
			count++
		}
	}
	return count
}
