package reader

import (
	"fmt"
	"os"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
)

// startStream - starts a stream over a real pipe. Returns the write end and the
// notification channel; both are cleaned up when the test ends.
func startStream(t *testing.T, height int) (Stream, *os.File, chan bool) {
	t.Helper()
	return startSmallStream(t, height, defaultCapacity, defaultChunkSize)
}

// startSmallStream - startStream holding `capacity` lines in memory and
// loading older ones in Chunks of `chunkSize` lines.
func startSmallStream(t *testing.T, height, capacity, chunkSize int) (Stream, *os.File, chan bool) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	next := make(chan bool)
	s := newStream(r, capacity, chunkSize)
	s.SetBufferSize(height)
	s.Start(next)

	t.Cleanup(func() {
		w.Close()
		s.Close()
	})
	return s, w, next
}

// numbers - the Line Number of each Line.
func numbers(lines []Line) string {
	each := make([]int, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Number)
	}
	return fmt.Sprint(each)
}

func Test_StreamLinesCarryTheirLineNumbers(t *testing.T) {
	s, w, next := startSmallStream(t, 2, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}

	testutil.Equal(t, "[18 19]", numbers(s.Lines())) // from the Live Buffer
	s.MoveBy(-100)
	testutil.Equal(t, "[0 1]", numbers(s.Lines())) // from a Chunk
}

func Test_StreamReadsLinesByLineNumberWhereverTheWindowIs(t *testing.T) {
	s, w, next := startSmallStream(t, 2, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}

	testutil.Equal(t, 20, s.Total())
	lines := s.Read(13, 17) // a Chunk, then the Live Buffer (holding 15..19)
	testutil.Equal(t, "[13 14 15 16]", numbers(lines))
	testutil.Equal(t, "line 13\nline 14\nline 15\nline 16", texts(lines))
	testutil.Equal(t, "[18 19]", numbers(s.Lines())) // the Window has not moved
}

func Test_StreamFirstLineMovesUpOnceLogFileFailsToRead(t *testing.T) {
	s, w, next := startSmallStream(t, 2, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}
	testutil.Equal(t, 0, s.First())

	s.(*stream).logFile.Close()
	testutil.Equal(t, "[15 16]", numbers(s.Read(0, 17)))
	testutil.Equal(t, 15, s.First())
}

func Test_StreamPinCentredOnLine(t *testing.T) {
	for _, test := range []struct {
		name     string
		height   int
		number   int
		expected string
	}{
		{name: "in the middle", height: 3, number: 10, expected: "[9 10 11]"},
		{name: "upper middle when height is even", height: 4, number: 10, expected: "[9 10 11 12]"},
		{name: "stops at the first line", height: 3, number: 0, expected: "[0 1 2]"},
		{name: "stops at the newest line", height: 3, number: 19, expected: "[17 18 19]"},
		{name: "in a Chunk", height: 3, number: 3, expected: "[2 3 4]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, w, next := startSmallStream(t, test.height, 5, 2)
			for i := range 20 {
				fmt.Fprintf(w, "line %d\n", i)
				<-next
			}

			s.PinCentredOn(test.number)
			fmt.Fprintln(w, "line 20")
			<-next

			testutil.Equal(t, test.expected, numbers(s.Lines())) // Pinned: line 20 is not shown
		})
	}
}

func Test_StreamWindowHasLineWhenNotified(t *testing.T) {
	s, w, next := startStream(t, 1)

	for i := range 200 {
		fmt.Fprintln(w, i)
		<-next
		testutil.Equal(t, fmt.Sprint(i), texts(s.Lines()))
	}
}

func Test_StreamPinnedWhileIngesting(t *testing.T) {
	s, w, next := startStream(t, 2)
	for i := range 10 {
		fmt.Fprintln(w, i)
		<-next
	}

	testutil.Equal(t, -3, s.MoveBy(-3))

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 10; i < 15; i += 1 {
			fmt.Fprintln(w, i)
			<-next
		}
	}()
	for {
		testutil.Equal(t, "5\n6", texts(s.Lines()))
		select {
		case <-done:
		default:
			continue
		}
		break
	}

	s.MoveToEnd()
	testutil.Equal(t, "13\n14", texts(s.Lines()))
}

func Test_StreamScrollsToFirstLineFromLogFile(t *testing.T) {
	s, w, next := startSmallStream(t, 2, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\r\n", i)
		<-next
	}

	testutil.Equal(t, -18, s.MoveBy(-100))
	testutil.Equal(t, "line 0\nline 1", texts(s.Lines()))
	testutil.Equal(t, 9, s.MoveBy(9))
	testutil.Equal(t, "line 9\nline 10", texts(s.Lines()))
}
