package reader

import (
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
)

func Test_LiveBufferTableDriven(t *testing.T) {
	for _, test := range []struct {
		name       string
		capacity   int
		push       []string
		total      int
		oldestHeld int
		from, to   int
		expected   string
	}{
		{
			name:       "empty",
			capacity:   3,
			push:       []string{},
			total:      0,
			oldestHeld: 0,
			from:       0,
			to:         0,
			expected:   "",
		},
		{
			name:       "under capacity",
			capacity:   3,
			push:       []string{"a", "b"},
			total:      2,
			oldestHeld: 0,
			from:       0,
			to:         2,
			expected:   "a\nb",
		},
		{
			name:       "wrapped keeps newest by line number",
			capacity:   3,
			push:       []string{"a", "b", "c", "d", "e"},
			total:      5,
			oldestHeld: 2,
			from:       2,
			to:         5,
			expected:   "c\nd\ne",
		},
		{
			name:       "sub range after wrap",
			capacity:   3,
			push:       []string{"a", "b", "c", "d", "e"},
			total:      5,
			oldestHeld: 2,
			from:       3,
			to:         4,
			expected:   "d",
		},
		{
			name:       "sub range after wrap twice",
			capacity:   2,
			push:       []string{"a", "b", "c", "d", "e"},
			total:      5,
			oldestHeld: 3,
			from:       3,
			to:         5,
			expected:   "d\ne",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			lb := NewLiveBuffer(test.capacity)
			for _, each := range test.push {
				lb.Push(Line{Text: each})
			}

			testutil.Equal(t, test.total, lb.Total())
			testutil.Equal(t, test.oldestHeld, lb.OldestHeld())
			testutil.Equal(t, test.expected, texts(lb.Lines(test.from, test.to)))
		})
	}
}
