package reader

// Window - the range of line numbers [start, end) of a LiveBuffer being shown.
// While following, the range is derived on read from the newest lines and
// start/end are not used (see docs/adr/0002).
type Window struct {
	lb        *LiveBuffer
	height    int
	start     int
	end       int
	following bool
}

// NewWindow - Creates a following window over `lb`.
func NewWindow(lb *LiveBuffer) *Window {
	return &Window{
		lb:        lb,
		following: true,
	}
}

// SetHeight - sets how many lines the window shows.
func (w *Window) SetHeight(height int) {
	w.height = height
}

// Following - `true` if the window is glued to the newest lines.
func (w *Window) Following() bool {
	return w.following
}

// Lines - returns the lines currently in the window.
func (w *Window) Lines() []string {
	w.clamp()
	if w.following {
		return w.lb.Lines(w.lastStart(), w.lb.Total())
	}
	return w.lb.Lines(w.start, w.end)
}

// MoveBy - scrolls the window by `n` lines (negative is up / older, positive is
// down / newer), stopping at either edge. Reaching the newest line switches the
// window to following. Returns the number of lines actually moved.
func (w *Window) MoveBy(n int) int {
	w.clamp()
	start := w.lastStart()
	if !w.following {
		start = w.start
	}

	last := w.lastStart()
	newStart := min(max(start+n, w.lb.OldestHeld()), last)

	if newStart == last {
		w.following = true
	} else {
		w.following = false
		w.start = newStart
		w.end = newStart + w.height
	}

	return newStart - start
}

// MoveToEnd - glues the window to the newest lines.
func (w *Window) MoveToEnd() {
	w.following = true
}

// ----------------------- PRIVATE

// clamp - shifts a pinned window forward to the oldest held line once its lines
// have been evicted. If that brings it to the newest line it follows again.
func (w *Window) clamp() {
	oldest := w.lb.OldestHeld()
	if w.following || w.start >= oldest {
		return
	}

	w.end += oldest - w.start
	w.start = oldest
	if w.end >= w.lb.Total() {
		w.following = true
	}
}

// lastStart - start of the window when it shows the newest lines.
func (w *Window) lastStart() int {
	return max(w.lb.OldestHeld(), w.lb.Total()-w.height)
}
