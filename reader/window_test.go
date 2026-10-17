package reader

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
)

// logFile - an in-memory Log File that grows as lines are written to it.
type logFile struct{ bytes.Buffer }

func (f *logFile) ReadAt(p []byte, off int64) (int, error) {
	return bytes.NewReader(f.Bytes()).ReadAt(p, off)
}

// unreadableFile - a Log File every read of which fails.
type unreadableFile struct{ logFile }

func (f *unreadableFile) ReadAt([]byte, int64) (int, error) {
	return 0, errors.New("unreadable")
}

// history - a History over an in-memory Log File, and that file.
type history struct {
	*History
	file *logFile
}

// newHistory - a History holding `capacity` lines in its Live Buffer and
// loading `chunkSize` lines per Chunk from an in-memory Log File.
func newHistory(capacity, chunkSize int) history {
	file := &logFile{}
	return history{NewHistory(capacity, chunkSize, file), file}
}

// pushRaw - writes lines "0".."n-1" to the Log File and the History, as the
// Stream does, so each line's text equals its line number.
func pushRaw(h history, n int) {
	for range n {
		raw := fmt.Sprintln(h.Total())
		h.file.WriteString(raw)
		h.Push(raw)
	}
}

func Test_WindowFollowing(t *testing.T) {
	for _, test := range []struct {
		name     string
		capacity int
		height   int
		pushed   int
		expected string
	}{
		{name: "empty", capacity: 10, height: 3, pushed: 0, expected: ""},
		{name: "fewer lines than height", capacity: 10, height: 3, pushed: 2, expected: "0\n1"},
		{name: "shows newest lines", capacity: 10, height: 3, pushed: 7, expected: "4\n5\n6"},
		{name: "keeps following after wrap", capacity: 4, height: 3, pushed: 9, expected: "6\n7\n8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newHistory(test.capacity, 2)
			w := NewWindow(h.History)
			w.SetHeight(test.height)
			pushRaw(h, test.pushed)

			testutil.Equal(t, true, w.Following())
			testutil.Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowPinned(t *testing.T) {
	for _, test := range []struct {
		name      string
		pushed    int
		moves     []int
		moved     int // return value of the last move
		pushAfter int
		following bool
		expected  string
	}{
		{name: "scroll up pins", pushed: 10, moves: []int{-2}, moved: -2, following: false, expected: "5\n6\n7"},
		{name: "pinned ignores new lines", pushed: 10, moves: []int{-2}, moved: -2, pushAfter: 5, following: false, expected: "5\n6\n7"},
		{name: "scroll down while following is a no-op", pushed: 10, moves: []int{3}, moved: 0, following: true, expected: "7\n8\n9"},
		{name: "scroll up stops at oldest", pushed: 10, moves: []int{-50}, moved: -7, following: false, expected: "0\n1\n2"},
		{name: "scroll down within pinned", pushed: 10, moves: []int{-5, 2}, moved: 2, following: false, expected: "4\n5\n6"},
		{name: "scroll down to newest re-follows", pushed: 10, moves: []int{-5, 9}, moved: 5, pushAfter: 2, following: true, expected: "9\n10\n11"},
		{name: "fewer lines than height cannot scroll", pushed: 2, moves: []int{-1}, moved: 0, following: true, expected: "0\n1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newHistory(20, 2)
			w := NewWindow(h.History)
			w.SetHeight(3)
			pushRaw(h, test.pushed)

			moved := 0
			for _, n := range test.moves {
				moved = w.MoveBy(n)
			}
			pushRaw(h, test.pushAfter)

			testutil.Equal(t, test.moved, moved)
			testutil.Equal(t, test.following, w.Following())
			testutil.Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowPinnedAfterEviction(t *testing.T) {
	for _, test := range []struct {
		name      string
		moves     []int // applied after the eviction
		moved     int
		following bool
		expected  string
	}{
		{name: "keeps its line numbers", following: false, expected: "0\n1"},
		{name: "scroll down from evicted lines", moves: []int{1}, moved: 1, following: false, expected: "1\n2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newHistory(5, 2)
			w := NewWindow(h.History)
			w.SetHeight(2)
			pushRaw(h, 5) // holds 0..4
			w.MoveBy(-3)  // pinned on 0,1
			pushRaw(h, 2) // evicts 0, 1

			moved := 0
			for _, n := range test.moves {
				moved = w.MoveBy(n)
			}

			testutil.Equal(t, test.moved, moved)
			testutil.Equal(t, test.following, w.Following())
			testutil.Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}

func Test_WindowScrollsPastOldestHeld(t *testing.T) {
	h := newHistory(5, 2)
	w := NewWindow(h.History)
	w.SetHeight(3)
	pushRaw(h, 20) // holds 15..19; 0..14 only in the Log File

	testutil.Equal(t, -17, w.MoveBy(-50))
	testutil.Equal(t, false, w.Following())
	testutil.Equal(t, "0\n1\n2", strings.Join(w.Lines(), "\n"))
}

func Test_WindowMoveToEnd(t *testing.T) {
	h := newHistory(20, 2)
	w := NewWindow(h.History)
	w.SetHeight(3)
	pushRaw(h, 10)
	w.MoveBy(-4)

	w.MoveToEnd()
	pushRaw(h, 1)

	testutil.Equal(t, true, w.Following())
	testutil.Equal(t, "8\n9\n10", strings.Join(w.Lines(), "\n"))
}

func Test_WindowScrollsDownFromFirstLineAcrossChunks(t *testing.T) {
	h := newHistory(5, 2)
	w := NewWindow(h.History)
	w.SetHeight(3)
	pushRaw(h, 20) // holds 15..19; 0..14 only in the Log File
	w.MoveBy(-50)

	for start := 1; start <= 17; start += 1 {
		testutil.Equal(t, 1, w.MoveBy(1))
		expected := fmt.Sprintf("%d\n%d\n%d", start, start+1, start+2)
		testutil.Equal(t, expected, strings.Join(w.Lines(), "\n"))
	}
	testutil.Equal(t, true, w.Following())
}

func Test_WindowStripsCRLF(t *testing.T) {
	h := newHistory(5, 2)
	w := NewWindow(h.History)
	w.SetHeight(3)
	for i := range 20 {
		raw := fmt.Sprintf("%d\r\n", i)
		h.file.WriteString(raw)
		h.Push(raw)
	}

	testutil.Equal(t, "17\n18\n19", strings.Join(w.Lines(), "\n"))
	w.MoveBy(-50)
	testutil.Equal(t, "0\n1\n2", strings.Join(w.Lines(), "\n"))
}

func Test_WindowWhenLogFileUnreadable(t *testing.T) {
	for _, test := range []struct {
		name       string
		pushed     int
		moves      []int
		pushAfter  int
		movesAfter []int
		moved      int // return value of the last move
		following  bool
		expected   string
	}{
		{name: "scroll up stops at oldest held", pushed: 20, moves: []int{-50}, moved: -2, following: false, expected: "15\n16\n17"},
		{name: "pinned on evicted lines shifts to oldest held", pushed: 5, moves: []int{-2}, moved: -2, pushAfter: 3, following: false, expected: "3\n4\n5"},
		{name: "scroll from evicted lines starts at oldest held", pushed: 5, moves: []int{-2}, pushAfter: 3, movesAfter: []int{1}, moved: 1, following: false, expected: "4\n5\n6"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := &unreadableFile{}
			h := history{NewHistory(5, 2, file), &file.logFile}
			w := NewWindow(h.History)
			w.SetHeight(3)
			pushRaw(h, test.pushed)

			moved := 0
			for _, n := range test.moves {
				moved = w.MoveBy(n)
			}
			pushRaw(h, test.pushAfter)
			for _, n := range test.movesAfter {
				moved = w.MoveBy(n)
			}

			testutil.Equal(t, test.moved, moved)
			testutil.Equal(t, test.following, w.Following())
			testutil.Equal(t, test.expected, strings.Join(w.Lines(), "\n"))
		})
	}
}
