// Package logfeed provides a real reader.Stream fed over a pipe, for UI and
// Executor tests, and the test doubles and Lines they share.
package logfeed

import (
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/reader"
)

// Feed - the write end of a Log Feed.
type Feed struct {
	t       *testing.T    // the test the Log Feed is written for
	stream  reader.Stream // the stream reading the Log Feed
	w       *os.File      // the pipe's write end
	next    chan bool     // drained by Write once Started; nil while the UI starts the stream
	written int           // Lines written so far
}

// New - a real reader.Stream over a pipe, not yet started: the UI starts it.
// Returns it with the Feed writing to it.
func New(t *testing.T) (reader.Stream, *Feed) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stream := reader.NewStream(r)
	// Ends the Log Feed; the stream stays open, as the UI may still be sent a
	// line. A failed close is safe to drop: the test has ended.
	t.Cleanup(func() { _ = w.Close() })
	return stream, &Feed{t: t, stream: stream, w: w}
}

// Started - a real reader.Stream over a pipe, started and fed `lines`. Returns
// it with the Feed writing to it.
func Started(t *testing.T, lines ...string) (reader.Stream, *Feed) {
	t.Helper()
	stream, feed := New(t)
	feed.next = make(chan bool)
	stream.Start(feed.next)
	t.Cleanup(stream.Close)

	feed.Write(lines...)
	return stream, feed
}

// Write - writes each of `lines` to the Log Feed. Once Started, waits for
// each to arrive in the stream, failing the test after a second. Otherwise
// the UI receives them as it reads them: the stream reads the next line only
// once the UI has taken the last, so lines that overfill the pipe (some
// 16 KB) block until it does.
func (f *Feed) Write(lines ...string) {
	f.t.Helper()
	for _, line := range lines {
		if _, err := fmt.Fprintln(f.w, line); err != nil {
			f.t.Fatal(err)
		}
		f.written++
		if f.next == nil {
			continue
		}
		select {
		case <-f.next:
		case <-time.After(time.Second):
			f.t.Fatalf("line %q did not arrive in the stream", line)
		}
	}
}

// Arrived - whether every Line written has arrived in the stream.
func (f *Feed) Arrived() bool {
	return f.stream.Total() >= f.written
}

// Numbered - Lines "ok 0".."ok n-1", but "err i" for each i in `errs`.
func Numbered(n int, errs ...int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprint("ok ", i)
		if slices.Contains(errs, i) {
			lines[i] = fmt.Sprint("err ", i)
		}
	}
	return lines
}
