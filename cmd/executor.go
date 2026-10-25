package cmd

import (
	"errors"
	"strings"
	"sync"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// Stream - the Window as the UI sees it: after every Pass.
type Stream interface {
	reader.Stream
	GetWindow() (shown, gutter string) // shown text and Gutter cells of the lines in the Window the Filter does not hide, after every Pass, each joined by newlines; while the Filter hides Lines, the Window takes in others to fill its height
	Update(msg any) error              // hands `msg` to every Pass, such as SearchFor
}

var _ Stream = (*Executor)(nil)

// Pass - processing run over each Line in the Window before it is shown,
// formatting its shown text. Update changes its state on a message of its own
// (such as SearchFor) and ignores any other (see docs/adr/0006).
type Pass interface {
	Run(lines []reader.Line) []reader.Line
	Update(msg any) error
}

// Executor - a reader.Stream whose Window has the Filter run over it on each
// read, then every other Pass, then the Search, then the Gutter (see
// docs/adr/0005, docs/adr/0009).
// It gives the Search the History as the Passes before it leave it, to step
// through (see docs/adr/0008).
type Executor struct {
	reader.Stream

	mu     sync.Mutex // guards the Passes: updated in the UI's Update, run in a tea.Cmd
	passes []Pass     // every Pass but the Search and the Gutter: the Filter first, then the others in order
	search *search    // run after `passes`, stepping through the History they leave
	gutter *gutter    // run last, over each Line as every other Pass leaves it
}

// NewExecutor - an Executor over `stream` running the Filter, then `others` in
// order, then the Search, whose Matches are styled by `renderer`, then the
// Gutter.
func NewExecutor(stream reader.Stream, renderer *lipgloss.Renderer, others ...Pass) *Executor {
	passes := append([]Pass{newFilter()}, others...)
	e := &Executor{Stream: stream, passes: passes, search: newSearch(renderer), gutter: newGutter()}
	e.search.attach(shownHistory{e})
	return e
}

// Update - hands `msg` to the Filter, then every other Pass, then the Search,
// then the Gutter, and returns their errors joined.
func (e *Executor) Update(msg any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var errs []error
	for _, pass := range e.passes {
		errs = append(errs, pass.Update(msg))
	}
	errs = append(errs, e.search.Update(msg), e.gutter.Update(msg))
	return errors.Join(errs...)
}

// Lines - the lines in the Window, after every Pass.
func (e *Executor) Lines() []reader.Line {
	lines := e.Stream.Lines()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.gutter.Run(e.search.Run(e.takeIn(e.runOthers(lines))))
}

// GetWindow - the shown text and the Gutter cells of the lines in the Window
// the Filter does not hide, after every Pass, each joined by newlines.
func (e *Executor) GetWindow() (shown, gutter string) {
	lines := e.Lines()
	each := make([]string, 0, len(lines))
	cells := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.Filter.Hide {
			continue
		}
		each = append(each, line.Shown)
		cells = append(cells, gutterCells(line.Gutter))
	}
	return strings.Join(each, "\n"), strings.Join(cells, "\n")
}

// ------------------------- Private

// maxTakenIn - the most Lines beyond the Window a read takes in to fill it
// while the Filter hides Lines: as many as the Live Buffer holds, so a Filter
// that rarely matches never reads the whole History on every render.
const maxTakenIn = reader.DefaultCapacity

// takeIn - `lines` after every Pass but the Search and the Gutter, with as many
// older Lines before them, then newer ones after them, run over by the same
// Passes, as it takes to show as many Lines as the Window's height; it stops
// at the first and the newest Line, once a read finds nothing, and after
// maxTakenIn Lines. Callers hold mu.
func (e *Executor) takeIn(lines []reader.Line) []reader.Line {
	first, total, height := e.Stream.First(), e.Stream.Total(), e.Stream.Height()
	budget := maxTakenIn
	for _, older := range []bool{true, false} {
		for {
			missing := min(height-shownCount(lines), budget) // reads no more Lines than it may show
			if missing <= 0 || len(lines) == 0 {
				return lines
			}
			from, to := lines[0].Number-missing, lines[0].Number
			if !older {
				from = lines[len(lines)-1].Number + 1
				to = from + missing
			}
			from, to = max(from, first), min(to, total)
			if from >= to { // at the first, or the newest, Line
				break
			}
			read := e.runOthers(e.Stream.Read(from, to))
			if len(read) == 0 { // the Log File has failed to read
				break
			}
			budget -= len(read)
			if older {
				lines = append(read, lines...)
			} else {
				lines = append(lines, read...)
			}
		}
	}
	return lines
}

// shownCount - how many of `lines` the Filter does not hide.
func shownCount(lines []reader.Line) int {
	count := 0
	for _, line := range lines {
		if !line.Filter.Hide {
			count++
		}
	}
	return count
}

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

// shownHistory - the History as the Passes before the Search leave it, given
// to the Search; called from its Update, with mu held.
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
