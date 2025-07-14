package reader

import "strings"

type Buffer interface {
	Stringify(string) string
	Push(string)
}

type buffer struct {
	buf    []string // buffer
	size   int      // max size of the buffer
	filled int      // currently filled size
	i      int      // start index of buffer
	j      int      // end index of buffer
}

// NewBuffer - Creates a new string buffer of the provided size.
func NewBuffer(size int) Buffer {
	return &buffer{
		buf:  make([]string, size),
		size: size,
		i:    0,
		j:    0,
	}
}

// Push - takes a string and pushes into the buffer. If the buffer is full, it
// replaces the oldest data with the newest data.
func (b *buffer) Push(msg string) {
	b.buf[b.j] = msg
	b.j = (b.j + 1) % b.size

	if b.j == b.i {
		b.i = (b.i + 1) % b.size
	}

	if b.filled < b.size {
		b.filled += 1
	}
}

// Stringify - joins the data in the buffer into a single string with the
// provided separator.
func (b *buffer) Stringify(separator string) string {
	if b.filled == 0 {
		return ""
	}

	x := b.i
	if b.filled == b.size {
		x -= 1
		if x < 0 {
			x = b.size - 1
		}
	}

	final := make([]string, b.filled)
	for i := 0; i < b.filled; i += 1 {
		final[i] = b.buf[(x+i)%b.size]
	}

	return strings.Join(final, separator)
}
