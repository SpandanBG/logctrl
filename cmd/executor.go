package cmd

import (
	"strings"
	"sync"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// Stream - the Window as the UI sees it: after every Pass.
type Stream interface {
	reader.Stream
	GetWindow() string         // shown text of the lines in the Window, after every Pass, joined by newlines
	Search(regex string) error // "" ends the Search
}

var _ Stream = (*Executor)(nil)

// Pass - processing run over each Line in the Window before it is shown,
// formatting its shown text.
type Pass interface {
	Run(lines []reader.Line) []reader.Line
}

// Executor - a reader.Stream whose Window has every Pass the user has
// triggered run over it on each read, the Search last (see docs/adr/0005).
type Executor struct {
	reader.Stream

	renderer *lipgloss.Renderer // styles each Pass
	mu       sync.Mutex         // guards the Passes: Search runs in Update, reads in a tea.Cmd
	passes   []Pass             // the Passes triggered, but the Search
	search   *Search            // nil while no Search is running
}

// NewExecutor - an Executor over `stream`, with no Pass triggered, whose
// Passes are styled by `renderer`.
func NewExecutor(stream reader.Stream, renderer *lipgloss.Renderer) *Executor {
	return &Executor{Stream: stream, renderer: renderer}
}

// Search - starts a Search for `regex`, or ends it if `regex` is empty. A regex
// that does not compile ends the Search and is returned as the error.
func (e *Executor) Search(regex string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.search = nil
	if regex == "" {
		return nil
	}

	search, err := NewSearch(e.renderer, regex)
	if err != nil {
		return err
	}
	e.search = search
	return nil
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
	if e.search != nil {
		lines = e.search.Run(lines)
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
