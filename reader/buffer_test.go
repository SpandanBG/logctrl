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

func TestBufferTableDriven(t *testing.T) {
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

func BenchmarkBuffer(b *testing.B) {
	buf := NewBuffer(2)

	for _, each := range []string{"a", "b", "c", "d"} {
		buf.Push(each)
	}

	_ = buf.Stringify("\n")
}
