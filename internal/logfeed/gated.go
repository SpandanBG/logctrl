package logfeed

import (
	"sync/atomic"

	"github.com/SpandanBG/logctrl/reader"
)

// maxWaiting - the most reads that can wait at once without blocking on
// Reading: far more than a step's scan makes at a time, one.
const maxWaiting = 100

// Gated - a reader.Stream telling a step's scan from a read of the Window
// itself by its reads of more Lines than its Window is tall. It counts them,
// and once armed makes them wait until released, so a test can act while a
// scan is running.
type Gated struct {
	reader.Stream
	Reading chan struct{} // receives once per read waiting

	window  int           // the most Lines a read of the Window takes
	scans   atomic.Int64  // reads of more than `window` Lines so far
	armed   atomic.Bool   // reads of more than `window` Lines wait
	release chan struct{} // closed to let waiting reads finish
}

// NewGated - `stream`, Gated, under a Window `window` Lines tall.
func NewGated(stream reader.Stream, window int) *Gated {
	return &Gated{Stream: stream, Reading: make(chan struct{}, maxWaiting), window: window, release: make(chan struct{})}
}

// Arm - makes each scan's read wait, receiving on Reading, until Release.
func (g *Gated) Arm() {
	g.armed.Store(true)
}

// Release - lets the waiting reads finish, and later ones read straight away.
func (g *Gated) Release() {
	g.armed.Store(false)
	close(g.release)
}

// Scans - the reads a step's scan has made so far.
func (g *Gated) Scans() int {
	return int(g.scans.Load())
}

// Read - the Lines in [from, to), as the stream reads them; a scan's read is
// counted, and waits while armed.
func (g *Gated) Read(from, to int) []reader.Line {
	if to-from > g.window {
		g.scans.Add(1)
		if g.armed.Load() {
			g.Reading <- struct{}{}
			<-g.release
		}
	}
	return g.Stream.Read(from, to)
}
