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
// read, the Search last (see docs/adr/0005).
type Executor struct {
	reader.Stream

	mu     sync.Mutex // guards the Passes: updated in the UI's Update, run in a tea.Cmd
	passes []Pass     // every Pass but the Search
	search *Search
}

// NewExecutor - an Executor over `stream` running `others` in order, then
// `search`.
func NewExecutor(stream reader.Stream, search *Search, others ...Pass) *Executor {
	return &Executor{Stream: stream, passes: others, search: search}
}

// Update - hands `msg` to every Pass, returning their errors joined.
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
	for i := range lines {
		lines[i].Shown = lines[i].Text
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, pass := range e.passes {
		lines = pass.Run(lines)
	}
	return e.search.Run(lines)
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
