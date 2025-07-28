package reader

import "strings"

type Buffer interface {
	Stringify(string) string
	Resize(int)
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

// Resize - updates and size of the buffer and makes sure the old data is present
func (b *buffer) Resize(newSize int) {
	if b.size > newSize {
		b.shrink(newSize)
	} else {
		b.enlarge(newSize)
	}
}

// Stringify - joins the data in the buffer into a single string with the
// provided separator.
func (b *buffer) Stringify(separator string) string {
	if b.filled == 0 {
		return ""
	}

	final := make([]string, b.filled)
	b.forEach(func(s string, i int) bool {
		final[i] = s
		return true
	})

	return strings.Join(final, separator)
}

// ----------------------- PRIVATE

// shrink - reduces the size of the buffer and keeps old data
// TODO - optimize iteration of old buffer to copy to new buffer instead of
// copying from start to end, we can just try and copy from the items that would
// be left in the final buffer.
func (b *buffer) shrink(newSize int) {
	if b.size < newSize {
		return
	}

	nb := NewBuffer(newSize)

	b.forEach(func(s string, _ int) bool {
		nb.Push(s)
		return true
	})

	copy(nb.(*buffer), b)
}

// enlarge - increases the size of the buffer and keeps old data
func (b *buffer) enlarge(newSize int) {
	if b.size > newSize {
		return
	}

	nb := NewBuffer(newSize)

	b.forEach(func(s string, _ int) bool {
		nb.Push(s)
		return true
	})

	copy(nb.(*buffer), b)
}

// forEach - Iterates through each item in the buffer and calls the passed
// callback with the value and index. If callback returns `false`, the iteration
// will be stopped at that point.
func (b *buffer) forEach(act func(string, int) bool) {
	x := b.i
	if b.filled == b.size {
		x -= 1
		if x < 0 {
			x = b.size - 1
		}
	}

	for i := 0; i < b.filled; i += 1 {
		if ok := act(b.buf[(x+i)%b.size], i); !ok {
			break
		}
	}
}

// copy - copies one buffer items to another
func copy(from, to *buffer) {
	to.buf = from.buf
	to.size = from.size
	to.filled = from.filled
	to.i = from.i
	to.j = from.j
}
