// Package headless is the Log Feed as any UI sees it: the shown text and
// Gutter cells of the Lines in the Window, a notice when they may have
// changed, and the messages that change them (see docs/adr/0010).
package headless

import (
	"strings"
	"sync"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/reader"
)

// GutterWidth - the cells in each Line's Gutter, as Window gives them.
const GutterWidth = reader.GutterWidth

// Logs - the Window of a cmd.Stream as a UI shows it.
type Logs struct {
	stream    cmd.Stream    // the Executor, running every Pass over the Window
	changed   chan struct{} // holds at most one notice, so notices the UI has not taken collapse; closed once the stream is
	mu        sync.Mutex    // guards closed, so no notice is sent once changed is closed
	closed    bool          // changed is closed
	closeOnce sync.Once     // closes the stream once, however often Close is called
}

// New - the Logs over `stream`, started, its Window one Line tall until the
// first Resize.
func New(stream cmd.Stream) *Logs {
	l := &Logs{stream: stream, changed: make(chan struct{}, 1)}
	next := make(chan bool)
	stream.SetBufferSize(1)
	stream.Start(next)
	go func() { // ends once the stream closes `next`, closing changed
		for range next {
			l.notify()
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		l.closed = true
		close(l.changed)
	}()
	return l
}

// Changed - notified when the Window may show something different: after a
// new Line arrives and after every Update. Notices not yet taken collapse
// into one. Closed once the Logs are.
func (l *Logs) Changed() <-chan struct{} {
	return l.changed
}

// Window - the shown text and the Gutter cells of each Line in the Window the
// Filter does not hide, after every Pass, from one read.
func (l *Logs) Window() (shown, gutter []string) {
	for _, line := range l.stream.Lines() {
		if line.Filter.Hide {
			continue
		}
		shown = append(shown, line.Shown)
		gutter = append(gutter, gutterCells(line.Gutter))
	}
	return shown, gutter
}

// Update - acts on `msg` if it moves the Window (cmd.Scroll, cmd.Resize,
// cmd.Follow) and hands it to every Pass, but for cmd.Resize, so resizing
// never cancels a step; returns their errors joined, then notifies. On
// cmd.Follow the Window Follows only once the Passes have it, so no step
// still scanning can Pin the Window after it.
func (l *Logs) Update(msg any) error {
	defer l.notify()
	switch msg := msg.(type) {
	case cmd.Resize:
		l.stream.SetBufferSize(msg.Height)
		return nil
	case cmd.Scroll:
		l.stream.MoveBy(msg.N)
	case cmd.Follow:
		err := l.stream.Update(msg)
		l.stream.MoveToEnd()
		return err
	}
	return l.stream.Update(msg)
}

// Close - stops reading the Log Feed; Changed closes soon after.
func (l *Logs) Close() {
	l.closeOnce.Do(l.stream.Close)
}

// ------------------------- Private

// notify - leaves a notice on `changed`, unless one is already waiting or it
// is closed.
func (l *Logs) notify() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	select {
	case l.changed <- struct{}{}:
	default:
	}
}

// gutterCells - the cells of a Gutter, an empty cell shown as a space.
func gutterCells(gutter [reader.GutterWidth]rune) string {
	var cells strings.Builder
	for _, cell := range gutter {
		if cell == 0 {
			cell = ' '
		}
		cells.WriteRune(cell)
	}
	return cells.String()
}
