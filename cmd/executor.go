package cmd

import (
	"errors"
	"strings"
	"sync"

	"github.com/SpandanBG/logctrl/reader"
)

// Stream - the Window as the UI sees it: after every Pass.
type Stream interface {
	reader.Stream
	GetWindow() string    // shown text of the lines in the Window, after every Pass, joined by newlines
	Update(msg any) error // hands `msg` to every Pass, such as SearchFor
}

var _ Stream = (*Executor)(nil)

// Pass - processing run over each Line in the Window before it is shown,
// formatting its shown text. Update changes its state on a message of its own
// (such as SearchFor) and ignores any other (see docs/adr/0006).
type Pass interface {
	Run(lines []reader.Line) []reader.Line
	Update(msg any) error
}

// Executor - a reader.Stream whose Window has every Pass run over it on each
// read, the Search last (see docs/adr/0005). It gives the Search the History
// as the other Passes leave it, to step through (see docs/adr/0008).
type Executor struct {
	reader.Stream

	mu     sync.Mutex // guards the Passes: updated in the UI's Update, run in a tea.Cmd
	passes []Pass     // every Pass but the Search
	search *Search
}

// NewExecutor - an Executor over `stream` running `others` in order, then
// `search`.
func NewExecutor(stream reader.Stream, search *Search, others ...Pass) *Executor {
	e := &Executor{Stream: stream, passes: others, search: search}
	search.attach(shownHistory{e})
	return e
}

// Update - hands `msg` to every Pass, the Search last, and returns their
// errors joined.
func (e *Executor) Update(msg any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var errs []error
	for _, pass := range e.passes {
		errs = append(errs, pass.Update(msg))
	}
	errs = append(errs, e.search.Update(msg))
	return errors.Join(errs...)
}

// Lines - the lines in the Window, after every Pass.
func (e *Executor) Lines() []reader.Line {
	lines := e.Stream.Lines()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.search.Run(e.runOthers(lines))
}

// runOthers - copies of `lines` after every Pass but the Search. Callers hold
// mu.
func (e *Executor) runOthers(lines []reader.Line) []reader.Line {
	for i := range lines {
		lines[i].Shown = lines[i].Text
	}
	for _, pass := range e.passes {
		lines = pass.Run(lines)
	}
	return lines
}

// GetWindow - the shown text of the lines in the Window, after every Pass,
// joined by newlines.
func (e *Executor) GetWindow() string {
	lines := e.Lines()
	shown := make([]string, 0, len(lines))
	for _, line := range lines {
		shown = append(shown, line.Shown)
	}
	return strings.Join(shown, "\n")
}

// shownHistory - the History as every Pass but the Search leaves it, given to
// the Search; called from its Update, with mu held.
type shownHistory struct {
	e *Executor
}

func (h shownHistory) Window() []int {
	lines := h.e.Stream.Lines()
	numbers := make([]int, 0, len(lines))
	for _, line := range lines {
		numbers = append(numbers, line.Number)
	}
	return numbers
}

func (h shownHistory) Total() int { return h.e.Stream.Total() }

func (h shownHistory) First() int { return h.e.Stream.First() }

// Read - reads with mu released, so the UI can render and send messages
// meanwhile.
func (h shownHistory) Read(from, to int) []reader.Line {
	h.e.mu.Unlock()
	lines := h.e.Stream.Read(from, to)
	h.e.mu.Lock()
	return h.e.runOthers(lines)
}

func (h shownHistory) PinCentredOn(number int) { h.e.Stream.PinCentredOn(number) }
