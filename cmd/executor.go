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
	GetWindow() string         // text of the lines in the Window, Matches highlighted, joined by newlines
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

	mu        sync.Mutex // guards the Passes: Search runs in Update, reads in a tea.Cmd
	search    *Search
	passes    []Pass
	highlight lipgloss.Style // how each Match is shown
}

// NewExecutor - an Executor over `stream`, running a Search, whose Matches
// are styled by `renderer`.
func NewExecutor(stream reader.Stream, renderer *lipgloss.Renderer) *Executor {
	search := NewSearch()
	return &Executor{
		Stream: stream,
		search: search,
		passes: []Pass{search},
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
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

// GetWindow - the text of the lines in the Window, after every Pass, with each
// Match highlighted, joined by newlines.
func (e *Executor) GetWindow() string {
	lines := e.Lines()
	texts := make([]string, 0, len(lines))
	for _, line := range lines {
		texts = append(texts, e.render(line))
	}
	return strings.Join(texts, "\n")
}

// render - the text of `line` with each of its Matches highlighted.
func (e *Executor) render(line reader.Line) string {
	var text strings.Builder
	from := 0
	for _, match := range line.Matches {
		text.WriteString(line.Text[from:match.Start])
		text.WriteString(e.highlight.Render(line.Text[match.Start:match.End]))
		from = match.End
	}
	text.WriteString(line.Text[from:])
	return text.String()
}
