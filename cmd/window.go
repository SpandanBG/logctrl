package cmd

import "github.com/SpandanBG/logctrl/reader"

// maxTakenIn - the most Lines beyond the Window a read takes in to fill it
// while the Filter hides Lines: as many as the Live Buffer holds, so a Filter
// that rarely matches never reads the whole History on every render.
const maxTakenIn = reader.DefaultCapacity

// window - the Lines being shown, read from the History through a
// reader.Stream: while Pinned, from its anchor, the Line Number its top Line
// is shown from, on. While Following, they are derived on read from the
// newest Lines and the anchor is not used (see docs/adr/0002).
type window struct {
	stream    reader.Stream                     // the History read from
	run       func([]reader.Line) []reader.Line // the Passes before the Search, run over the Lines it takes in; called with the Executor's lock held
	height    int                               // the Lines shown, as tall as the Log View
	anchor    int                               // Line Number the top Line is shown from, while Pinned: the first the Filter does not hide at or after it
	following bool                              // shows the newest Lines, whatever the anchor is
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
		to := w.stream.Total() // read once, so a Line arriving meanwhile is not taken in
		from := max(w.stream.First(), to-w.height)
		if !w.following {
			from, to = w.anchor, min(w.anchor+w.height, to)
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

// moveBy - Scrolls the window `n` of the Lines the Filter does not hide
// (negative is up / older, positive is down / newer), from its top shown Line,
// stopping at either edge. Its bottom reaching the newest shown Line makes it
// Following.
func (w *window) moveBy(n int) {
	shown := w.shown() // clamps the window if its Lines can no longer be read
	from, ok := topShown(shown)
	pinnedFromAnchor := !w.following && (!ok || w.anchor < from) // the top shown Line is beyond the anchor, or there is none
	switch {
	case pinnedFromAnchor:
		from = w.anchor
	case !ok:
		from = w.newestAnchor()
	}

	anchor := w.newer(from, n)
	if n < 0 {
		anchor = w.older(from, -n)
	}
	w.following = false
	w.anchor = anchor
	if w.reachesNewest(w.height) {
		w.following = true
	}
}

// pinCentredOn - Pins the window with line `number` in the middle of the
// Lines the Filter does not hide (the upper middle row if its height is
// even), stopping at the first and newest of them, and looking back at most
// maxTakenIn Lines.
func (w *window) pinCentredOn(number int) {
	w.following = false
	w.anchor = w.older(number, (w.height-1)/2)
	if w.reachesNewest(w.height - 1) { // not full: stops at the newest shown Line
		w.anchor = w.older(w.stream.Total(), w.height)
	}
}

// follow - makes the window Following.
func (w *window) follow() {
	w.following = true
}

// clamp - shifts a Pinned window forward to the first Line that can be read
// (see reader.Stream's First). If that brings its bottom to the newest Line
// the Filter does not hide it Follows again.
func (w *window) clamp() {
	first := w.stream.First()
	if w.following || w.anchor >= first {
		return
	}

	w.anchor = first
	if w.reachesNewest(w.height) {
		w.following = true
	}
}

// newestAnchor - the anchor of the window when it shows the newest Lines.
func (w *window) newestAnchor() int {
	return max(w.stream.First(), w.stream.Total()-w.height)
}

// takeIn - `lines`, with as many newer Lines after them, then older ones
// before them, run over by `run`, as it takes to show as many Lines as the
// window's height; it stops at the newest and the first Line, once a read
// finds nothing, and after maxTakenIn Lines. Each read is twice as long as the
// last, and the Lines read past the last one shown are left out. Callers hold
// the Executor's lock.
func (w *window) takeIn(lines []reader.Line) []reader.Line {
	first, total, height := w.stream.First(), w.stream.Total(), w.height
	budget := maxTakenIn
	for _, back := range []bool{false, true} {
		for span := height; ; span *= 2 {
			missing := height - shownCount(lines)
			if missing <= 0 || budget <= 0 || len(lines) == 0 {
				return lines
			}
			span = min(max(span, missing), budget)
			from, to := lines[0].Number-span, lines[0].Number
			if !back {
				from = lines[len(lines)-1].Number + 1
				to = from + span
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
			if back {
				lines = append(newestShown(read, missing), lines...)
			} else {
				lines = append(lines, oldestShown(read, missing)...)
			}
		}
	}
	return lines
}

// oldestShown - `lines` up to the `n`-th the Filter does not hide; all of them
// if it hides more.
func oldestShown(lines []reader.Line, n int) []reader.Line {
	for i, line := range lines {
		if !line.Filter.Hide {
			if n--; n == 0 {
				return lines[:i+1]
			}
		}
	}
	return lines
}

// newestShown - `lines` from the `n`-th the Filter does not hide, counting
// back from the newest; all of them if it hides more.
func newestShown(lines []reader.Line, n int) []reader.Line {
	for i := len(lines) - 1; i >= 0; i-- {
		if !lines[i].Filter.Hide {
			if n--; n == 0 {
				return lines[i:]
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

// older - the Line Number of the `n`-th Line the Filter does not hide before line
// `from`; the first Line if there are fewer, or where the Lines read stop
// after maxTakenIn of them (or `n`, if more), or once a read finds nothing.
func (w *window) older(from, n int) int {
	to := from
	budget := max(maxTakenIn, n)
	for span := max(n, w.height); n > 0; span *= 2 {
		first := w.stream.First() // moves up once a read finds the Log File failed
		if to <= first {
			return first
		}
		if budget <= 0 {
			return to
		}
		span = min(span, budget)
		lines := w.run(w.stream.Read(max(to-span, first), to))
		if len(lines) == 0 {
			return to
		}
		budget -= len(lines)
		for i := len(lines) - 1; i >= 0; i-- {
			if !lines[i].Filter.Hide {
				if n--; n == 0 {
					return lines[i].Number
				}
			}
		}
		to = lines[0].Number
	}
	return to
}

// newer - the Line Number of the `n`-th Line the Filter does not hide after the
// first at or after line `from`; the newest Line Number plus one if there are
// fewer, or where the Lines read stop after maxTakenIn of them (or `n`, if
// more), or once a read finds nothing.
func (w *window) newer(from, n int) int {
	n++ // the first shown Line at or after `from` is the top one
	budget := max(maxTakenIn, n)
	for span := max(n, w.height); ; span *= 2 {
		total := w.stream.Total()
		if from >= total {
			return total
		}
		if budget <= 0 {
			return from
		}
		span = min(span, budget)
		lines := w.run(w.stream.Read(from, min(from+span, total)))
		if len(lines) == 0 {
			return from
		}
		budget -= len(lines)
		for _, line := range lines {
			if !line.Filter.Hide {
				if n--; n == 0 {
					return line.Number
				}
			}
		}
		from = lines[len(lines)-1].Number + 1
	}
}

// reachesNewest - whether no more than `n` of the Lines the Filter does not
// hide lie from the anchor to the newest Line, within maxTakenIn Lines.
func (w *window) reachesNewest(n int) bool {
	return w.newer(w.anchor, n) >= w.stream.Total()
}

// topShown - the Line Number of the first of `lines` the Filter does not
// hide; `false` if it hides them all.
func topShown(lines []reader.Line) (int, bool) {
	for _, line := range lines {
		if !line.Filter.Hide {
			return line.Number, true
		}
	}
	return 0, false
}
