package cmd

import (
	"errors"
	"sync"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// Stream - the Window after every Pass: its Lines() have every Pass run over
// them and, while the Filter hides Lines, take in others to fill its height.
type Stream interface {
	Start(next chan bool) // starts reading the Log Feed, notifying `next` as each Line arrives
	Lines() []reader.Line // the Lines in the Window, after every Pass
	Update(msg any) error // moves the Window (Scroll, Resize, Follow) and hands `msg` to every Pass, such as SearchFor
	Close()               // stops reading the Log Feed
}

var _ Stream = (*Executor)(nil)

// Scroll - the message Scrolling the Window N Lines: negative is up (older),
// positive is down (newer).
type Scroll struct {
	N int // the Lines to Scroll: negative is up, positive is down
}

// Resize - the message making the Window Height Lines tall.
type Resize struct {
	Height int // the Lines the Window shows
}

// Pass - processing run over each Line in the Window before it is shown,
// formatting its shown text. Update changes its state on a message of its own
// (such as SearchFor) and ignores any other (see docs/adr/0006).
type Pass interface {
	Run(lines []reader.Line) []reader.Line
	Update(msg any) error
}

// Executor - the Window over the History of a reader.Stream, with the Filter
// run over it on each read, then every other Pass, then the Search, then the
// Gutter (see docs/adr/0005, docs/adr/0009, docs/adr/0011).
// It gives the Search the History as the Passes before it leave it, to step
// through (see docs/adr/0008).
type Executor struct {
	stream reader.Stream // the History of the Log Feed

	mu     sync.Mutex // guards the Window and the Passes: updated in the UI's Update, read in a tea.Cmd
	window *window    // the Lines shown, taking in others while the Filter hides some
	passes []Pass     // every Pass but the Search and the Gutter: the Filter first, then the others in order
	search *search    // run after `passes`, stepping through the History they leave
	gutter *gutter    // run last, over each Line as every other Pass leaves it
}

// NewExecutor - an Executor over `stream` running the Filter, then `others` in
// order, then the Search, whose Matches are styled by `renderer`, then the
// Gutter.
func NewExecutor(stream reader.Stream, renderer *lipgloss.Renderer, others ...Pass) *Executor {
	passes := append([]Pass{newFilter()}, others...)
	e := &Executor{stream: stream, passes: passes, search: newSearch(renderer), gutter: newGutter()}
	e.window = newWindow(stream, e.runOthers)
	e.search.attach(shownHistory{e})
	return e
}

// Update - hands `msg` to the Filter, then every other Pass, then the Search,
// then the Gutter, and returns their errors joined. Resize only resizes the
// Window, never reaching the Passes, so it cancels no step. Scroll and Follow
// then move the Window, under the same lock as the Passes, so no step still
// scanning can Pin the Window after a Follow.
func (e *Executor) Update(msg any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch msg := msg.(type) {
	case Resize:
		e.window.resize(msg.Height)
		return nil
	case Scroll:
		defer e.window.moveBy(msg.N)
	case Follow:
		defer e.window.follow()
	}

	var errs []error
	for _, pass := range e.passes {
		errs = append(errs, pass.Update(msg))
	}
	errs = append(errs, e.search.Update(msg), e.gutter.Update(msg))
	return errors.Join(errs...)
}

// Start - starts reading the Log Feed, notifying `next` as each Line arrives.
func (e *Executor) Start(next chan bool) {
	e.stream.Start(next)
}

// Close - stops reading the Log Feed.
func (e *Executor) Close() {
	e.stream.Close()
}

// Lines - the lines in the Window, after every Pass.
func (e *Executor) Lines() []reader.Line {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.gutter.Run(e.search.Run(e.window.shown()))
}

// shownHistory - the History as the Passes before the Search leave it, given
// to the Search; called from its Update, with mu held.
type shownHistory struct {
	e *Executor // the Executor whose History and Window the Search sees
}

// Window - the Line Numbers of the Lines in the Window.
func (h shownHistory) Window() []int {
	lines := h.e.window.lines()
	numbers := make([]int, 0, len(lines))
	for _, line := range lines {
		numbers = append(numbers, line.Number)
	}
	return numbers
}

// Total - the lines received so far.
func (h shownHistory) Total() int { return h.e.stream.Total() }

// First - the Line Number of the oldest line that can be read.
func (h shownHistory) First() int { return h.e.stream.First() }

// Read - reads with mu released, so the UI can render and send messages
// meanwhile.
func (h shownHistory) Read(from, to int) []reader.Line {
	h.e.mu.Unlock()
	lines := h.e.stream.Read(from, to)
	h.e.mu.Lock()
	return h.e.runOthers(lines)
}

// PinCentredOn - Pins the Window centred on line `number`.
func (h shownHistory) PinCentredOn(number int) { h.e.window.pinCentredOn(number) }

// Follow - makes the Window Following.
func (h shownHistory) Follow() { h.e.window.follow() }

// ------------------------- Private

// runOthers - copies of `lines` after every Pass but the Search and the
// Gutter, the Filter first. Callers hold mu.
func (e *Executor) runOthers(lines []reader.Line) []reader.Line {
	for i := range lines {
		lines[i].Shown = lines[i].Text
	}
	for _, pass := range e.passes {
		lines = pass.Run(lines)
	}
	return lines
}
