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

// ReadAt - reads what has been written, as from a file.
func (f *logFile) ReadAt(p []byte, off int64) (int, error) {
	return bytes.NewReader(f.Bytes()).ReadAt(p, off)
}

// unreadableFile - a Log File every read of which fails.
type unreadableFile struct{ logFile }

// ReadAt - fails.
func (f *unreadableFile) ReadAt([]byte, int64) (int, error) {
	return 0, errors.New("unreadable")
}

// texts - the text of each Line, joined by newlines.
func texts(lines []Line) string {
	each := make([]string, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Text)
	}
	return strings.Join(each, "\n")
}

// history - a History over an in-memory Log File, and that file.
type history struct {
	*History          // the History under test
	file     *logFile // its Log File, written as lines are pushed
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

// lines - the text of the Lines of `h` in [from, to), joined by newlines,
// failing the test if the Log File fails to read.
func lines(t *testing.T, h history, from, to int) string {
	t.Helper()
	read, ok := h.Lines(from, to)
	if !ok {
		t.Fatalf("lines %d to %d did not read", from, to)
	}
	return texts(read)
}

func Test_HistoryReadsNewestLines(t *testing.T) {
	for _, test := range []struct {
		name     string
		capacity int
		pushed   int
		expected string
	}{
		{name: "no lines yet", capacity: 10, pushed: 0, expected: ""},
		{name: "the newest lines", capacity: 10, pushed: 7, expected: "4\n5\n6"},
		{name: "once the Live Buffer wraps", capacity: 4, pushed: 9, expected: "6\n7\n8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := newHistory(test.capacity, 2)
			pushRaw(h, test.pushed)

			testutil.Equal(t, test.expected, lines(t, h, max(test.pushed-3, 0), test.pushed))
		})
	}
}

func Test_HistoryReadsLinesPastOldestHeldFromLogFile(t *testing.T) {
	h := newHistory(5, 2)
	pushRaw(h, 20) // holds 15..19; 0..14 only in the Log File

	testutil.Equal(t, "0\n1\n2", lines(t, h, 0, 3))
	for from := 1; from <= 17; from += 1 { // across Chunks, and into the Live Buffer
		expected := fmt.Sprintf("%d\n%d\n%d", from, from+1, from+2)
		testutil.Equal(t, expected, lines(t, h, from, from+3))
	}
}

func Test_HistoryKeepsLineNumbersOfLinesLeavingLiveBuffer(t *testing.T) {
	h := newHistory(5, 2)
	pushRaw(h, 5) // holds 0..4

	pushRaw(h, 2) // 0 and 1 leave the Live Buffer

	testutil.Equal(t, "0\n1", lines(t, h, 0, 2))
}

func Test_HistoryStripsCRLF(t *testing.T) {
	h := newHistory(5, 2)
	for i := range 20 {
		raw := fmt.Sprintf("%d\r\n", i)
		h.file.WriteString(raw)
		h.Push(raw)
	}

	testutil.Equal(t, "17\n18\n19", lines(t, h, 17, 20)) // from the Live Buffer
	testutil.Equal(t, "0\n1\n2", lines(t, h, 0, 3))      // from a Chunk
}

func Test_HistoryFirstMovesUpToOldestHeldOnceLogFileFailsToRead(t *testing.T) {
	file := &unreadableFile{}
	h := history{NewHistory(5, 2, file), &file.logFile}
	pushRaw(h, 20) // holds 15..19
	testutil.Equal(t, 0, h.First())

	_, ok := h.Lines(0, 3)

	testutil.Equal(t, false, ok)
	testutil.Equal(t, 15, h.First())
	testutil.Equal(t, "15\n16\n17", lines(t, h, 15, 18))
}
