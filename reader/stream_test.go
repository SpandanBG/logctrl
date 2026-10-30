package reader

import (
	"fmt"
	"os"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
)

// startStream - starts a stream over a real pipe. Returns the write end and the
// notification channel; both are cleaned up when the test ends.
func startStream(t *testing.T) (Stream, *os.File, chan bool) {
	t.Helper()
	return startSmallStream(t, DefaultCapacity, defaultChunkSize)
}

// startSmallStream - startStream holding `capacity` lines in memory and
// loading older ones in Chunks of `chunkSize` lines.
func startSmallStream(t *testing.T, capacity, chunkSize int) (Stream, *os.File, chan bool) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	next := make(chan bool)
	s := newStream(r, capacity, chunkSize)
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
	s, w, next := startSmallStream(t, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}

	testutil.Equal(t, "[18 19]", numbers(s.Read(18, 20))) // from the Live Buffer
	testutil.Equal(t, "[0 1]", numbers(s.Read(0, 2)))     // from a Chunk
}

func Test_StreamReadsLinesByLineNumber(t *testing.T) {
	s, w, next := startSmallStream(t, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}

	testutil.Equal(t, 20, s.Total())
	lines := s.Read(13, 17) // a Chunk, then the Live Buffer (holding 15..19)
	testutil.Equal(t, "[13 14 15 16]", numbers(lines))
	testutil.Equal(t, "line 13\nline 14\nline 15\nline 16", texts(lines))
}

func Test_StreamFirstLineMovesUpOnceLogFileFailsToRead(t *testing.T) {
	s, w, next := startSmallStream(t, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\n", i)
		<-next
	}
	testutil.Equal(t, 0, s.First())

	s.(*stream).logFile.Close()
	testutil.Equal(t, "[15 16]", numbers(s.Read(0, 17)))
	testutil.Equal(t, 15, s.First())
}

func Test_StreamHasLineWhenNotified(t *testing.T) {
	s, w, next := startStream(t)

	for i := range 200 {
		fmt.Fprintln(w, i)
		<-next
		testutil.Equal(t, fmt.Sprint(i), texts(s.Read(i, i+1)))
	}
}

func Test_StreamReadsSameLinesWhileIngesting(t *testing.T) {
	s, w, next := startStream(t)
	for i := range 10 {
		fmt.Fprintln(w, i)
		<-next
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 10; i < 15; i += 1 {
			fmt.Fprintln(w, i)
			<-next
		}
	}()
	for {
		testutil.Equal(t, "5\n6", texts(s.Read(5, 7)))
		select {
		case <-done:
		default:
			continue
		}
		break
	}

	testutil.Equal(t, "13\n14", texts(s.Read(13, 15)))
}

func Test_StreamReadsFirstLinesBackFromLogFile(t *testing.T) {
	s, w, next := startSmallStream(t, 5, 2)
	for i := range 20 {
		fmt.Fprintf(w, "line %d\r\n", i)
		<-next
	}

	testutil.Equal(t, "line 0\nline 1", texts(s.Read(0, 2)))
	testutil.Equal(t, "line 9\nline 10", texts(s.Read(9, 11)))
}
