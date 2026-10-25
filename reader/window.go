package reader

// Window - the range of line numbers [start, end) of a LiveBuffer being shown.
// While following, the range is derived on read from the newest lines and
// start/end are not used (see docs/adr/0002).
type Window struct {
	h         *History
	height    int
	start     int
	end       int
	following bool
}

// NewWindow - Creates a following window over `h`.
func NewWindow(h *History) *Window {
	return &Window{
		h:         h,
		following: true,
	}
}

// SetHeight - sets how many lines the window shows.
func (w *Window) SetHeight(height int) {
	w.height = height
}

// Height - how many lines the window shows.
func (w *Window) Height() int {
	return w.height
}

// Following - `true` if the window is glued to the newest lines.
func (w *Window) Following() bool {
	return w.following
}

// Lines - returns the lines currently in the window.
func (w *Window) Lines() []Line {
	w.clamp()
	from, to := w.start, w.end
	if w.following {
		from, to = w.lastStart(), w.h.Total()
	}

	lines, ok := w.h.Lines(from, to)
	if !ok {
		return w.Lines() // the History now starts at its oldest held line
	}
	return lines
}

// MoveBy - scrolls the window by `n` lines (negative is up / older, positive is
// down / newer), stopping at either edge. Reaching the newest line switches the
// window to following. Returns the number of lines actually moved.
func (w *Window) MoveBy(n int) int {
	w.Lines() // clamps the window if its lines can no longer be read
	start := w.lastStart()
	if !w.following {
		start = w.start
	}

	newStart := w.reachable(start + n)
	if newStart == w.lastStart() {
		w.following = true
	} else {
		w.following = false
		w.start = newStart
		w.end = newStart + w.height
	}

	return newStart - start
}

// PinCentredOn - pins the window with line `number` in its middle (the upper
// middle row if its height is even), stopping at the first and newest lines.
func (w *Window) PinCentredOn(number int) {
	w.following = false
	w.start = w.reachable(number - (w.height-1)/2)
	w.end = w.start + w.height
}

// MoveToEnd - glues the window to the newest lines.
func (w *Window) MoveToEnd() {
	w.following = true
}

// ----------------------- PRIVATE

// reachable - the window start nearest `start` whose lines can be read.
func (w *Window) reachable(start int) int {
	start = min(max(start, w.h.First()), w.lastStart())
	if _, ok := w.h.Lines(start, min(start+w.height, w.h.Total())); !ok {
		return w.reachable(start) // the History now starts at its oldest held line
	}
	return start
}

// clamp - shifts a pinned window forward to the first line the History can
// show (see History.First). If that brings it to the newest line it follows
// again.
func (w *Window) clamp() {
	first := w.h.First()
	if w.following || w.start >= first {
		return
	}

	w.end += first - w.start
	w.start = first
	if w.end >= w.h.Total() {
		w.following = true
	}
}

// lastStart - start of the window when it shows the newest lines.
func (w *Window) lastStart() int {
	return max(w.h.First(), w.h.Total()-w.height)
}
