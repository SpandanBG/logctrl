package reader

// LiveBuffer - holds the most recent `capacity` lines of the log feed,
// addressable by their line number (position in the feed since start).
type LiveBuffer struct {
	buf   []Line // ring storage; line n lives at buf[n % capacity]
	total int    // number of lines ever pushed
}

// NewLiveBuffer - Creates a new live buffer holding at most `capacity` lines.
func NewLiveBuffer(capacity int) *LiveBuffer {
	return &LiveBuffer{
		buf: make([]Line, capacity),
	}
}

// Push - appends a line, evicting the oldest one when full.
func (lb *LiveBuffer) Push(line Line) {
	lb.buf[lb.total%len(lb.buf)] = line
	lb.total += 1
}

// Total - number of lines received so far.
func (lb *LiveBuffer) Total() int {
	return lb.total
}

// OldestHeld - line number of the oldest line still in the buffer.
func (lb *LiveBuffer) OldestHeld() int {
	return max(0, lb.total-len(lb.buf))
}

// Lines - returns the held lines with line numbers in [from, to). Callers must
// keep the range within [OldestHeld(), Total()].
func (lb *LiveBuffer) Lines(from, to int) []Line {
	lines := make([]Line, 0, to-from)
	for n := from; n < to; n += 1 {
		lines = append(lines, lb.buf[n%len(lb.buf)])
	}
	return lines
}
