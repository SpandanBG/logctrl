package cmd

import (
	"strings"
	"sync"

	"github.com/SpandanBG/logctrl/reader"
)

// Stream - the Window as the UI sees it: after every Pass.
type Stream interface {
	reader.Stream
	GetWindow() string         // text of the lines in the Window, joined by newlines
	Search(regex string) error // "" ends the Search
}

var _ Stream = (*Executor)(nil)

// Pass - processing run over each Line in the Window before it is shown.
type Pass interface {
	Run(lines []reader.Line) []reader.Line
}

// Executor - a reader.Stream whose Window has every Pass run over it on each
// read (see docs/adr/0005).
type Executor struct {
	reader.Stream

	mu     sync.Mutex // guards the Passes: Search runs in Update, reads in a tea.Cmd
	search *Search
	passes []Pass
}

// NewExecutor - an Executor over `stream`, running a Search.
func NewExecutor(stream reader.Stream) *Executor {
	search := NewSearch()
	return &Executor{
		Stream: stream,
		search: search,
		passes: []Pass{search},
	}
}

// Search - starts a Search for `regex`, or ends it if `regex` is empty. A regex
// that does not compile ends the Search and is returned as the error.
func (e *Executor) Search(regex string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.search.Set(regex)
}

// Lines - the lines in the Window, after every Pass.
func (e *Executor) Lines() []reader.Line {
	lines := e.Stream.Lines()

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, pass := range e.passes {
		lines = pass.Run(lines)
	}
	return lines
}

// GetWindow - the text of the lines in the Window, after every Pass, joined by
// newlines.
func (e *Executor) GetWindow() string {
	lines := e.Lines()
	texts := make([]string, 0, len(lines))
	for _, line := range lines {
		texts = append(texts, line.Text)
	}
	return strings.Join(texts, "\n")
}
