package reader

import "testing"

func Equal[T comparable](t *testing.T, expected, actual T) {
	if expected == actual {
		return
	}
	t.Errorf(
		"expected equal\n\texpected:\t%v\n\tactual:\t%v",
		expected, actual,
	)
}

func Test_BufferTableDriven(t *testing.T) {
	for _, test := range []struct {
		name     string
		size     int
		push     []string
		expected string
	}{
		{
			name:     "zero item",
			size:     1,
			push:     []string{},
			expected: "",
		},
		{
			name:     "single item",
			size:     1,
			push:     []string{"a"},
			expected: "a",
		},
		{
			name:     "2 items in 1 sized buffer",
			size:     1,
			push:     []string{"a", "b"},
			expected: "b",
		},
		{
			name:     "2 items in 3 sized buffer",
			size:     3,
			push:     []string{"a", "b"},
			expected: "a\nb",
		},
		{
			name:     "3 items in 2 sized buffer",
			size:     2,
			push:     []string{"a", "b", "c"},
			expected: "b\nc",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			buf := NewBuffer(test.size)
			for _, each := range test.push {
				buf.Push(each)
			}

			str := buf.Stringify("\n")

			Equal(t, test.expected, str)
		})
	}
}

func Test_BufferResizeTableDriven(t *testing.T) {
	for _, test := range []struct {
		name       string
		size       int
		resize     int
		push       []string
		expected   string
		rePush     []string
		reExpected string
	}{
		{
			name:       "enlarge from 1 to 2 - full",
			size:       1,
			resize:     2,
			push:       []string{"a"},
			expected:   "a",
			rePush:     []string{"b"},
			reExpected: "a\nb",
		},
		{
			name:       "enlarge from 2 to 4 - not full",
			size:       2,
			resize:     4,
			push:       []string{"a"},
			expected:   "a",
			rePush:     []string{"b", "c"},
			reExpected: "a\nb\nc",
		},
		{
			name:       "enlarge from 2 to 4 - overfill",
			size:       2,
			resize:     4,
			push:       []string{"a", "b", "c"},
			expected:   "b\nc",
			rePush:     []string{"d", "e"},
			reExpected: "b\nc\nd\ne",
		},
		{
			name:       "shrink from 4 to 2 - not full",
			size:       4,
			resize:     2,
			push:       []string{"a"},
			expected:   "a",
			rePush:     []string{"b"},
			reExpected: "a\nb",
		},
		{
			name:       "shrink from 4 to 2 - full",
			size:       4,
			resize:     2,
			push:       []string{"a", "b", "c", "d"},
			expected:   "a\nb\nc\nd",
			rePush:     []string{"e"},
			reExpected: "d\ne",
		},
		{
			name:       "shrink from 4 to 2 - overfill",
			size:       4,
			resize:     2,
			push:       []string{"a", "b", "c", "d", "e"},
			expected:   "b\nc\nd\ne",
			rePush:     []string{"f"},
			reExpected: "e\nf",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			buf := NewBuffer(test.size)

			// -------- pre-resize
			for _, each := range test.push {
				buf.Push(each)
			}

			str := buf.Stringify("\n")
			Equal(t, test.expected, str)

			// -------- post-resize
			buf.Resize(test.resize)
			for _, each := range test.rePush {
				buf.Push(each)
			}

			str = buf.Stringify("\n")
			Equal(t, test.reExpected, str)
		})
	}
}

func Benchmark_Buffer(b *testing.B) {
	buf := NewBuffer(4)

	for _, each := range []string{"a", "b", "c", "d"} {
		buf.Push(each)
	}

	buf.Resize(2)
	_ = buf.Stringify("\n")
}
