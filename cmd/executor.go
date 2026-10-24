package cmd

import (
	"errors"
	"slices"
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

// scanBatch - Lines read from the History at a time while looking for a Line
// with a Match outside the Window: a Chunk's worth (see docs/adr/0007).
const scanBatch = 1_000

// Pass - processing run over each Line in the Window before it is shown,
// formatting its shown text. Update changes its state on a message of its own
// (such as SearchFor) and ignores any other (see docs/adr/0006).
type Pass interface {
	Run(lines []reader.Line) []reader.Line
	Update(msg any) error
}

// Executor - a reader.Stream whose Window has every Pass run over it on each
// read, the Search last (see docs/adr/0005). It steps between Lines with a
// Match itself, scanning the History (see docs/adr/0007).
type Executor struct {
	reader.Stream

	mu       sync.Mutex // guards all below: updated in the UI's Update, run in a tea.Cmd
	passes   []Pass     // every Pass but the Search
	search   *Search
	none     int  // while there is no Current Line, the newest Line known to have no Match
	gen      int  // counts the messages handed to the Passes; a scan from an older one is cancelled
	scanning bool // a scan for message `gen` is running
}

// NewExecutor - an Executor over `stream` running `others` in order, then
// `search`.
func NewExecutor(stream reader.Stream, search *Search, others ...Pass) *Executor {
	return &Executor{Stream: stream, passes: others, search: search}
}

// PreviousMatch - the message stepping from the Current Line to the previous
// (older) Line with a Match. The Executor handles it, not the Passes (see
// docs/adr/0007).
type PreviousMatch struct{}

// NextMatch - the message stepping from the Current Line to the next (newer)
// Line with a Match. The Executor handles it, not the Passes.
type NextMatch struct{}

// FindCurrentLine - the message finding the Current Line once a SearchFor has
// started a Search: the newest Line with a Match at or above the Window's
// bottom Line. The Executor handles it, not the Passes; the UI sends it in the
// background, as it may scan the History.
type FindCurrentLine struct{}

// Update - finds the Current Line on FindCurrentLine and steps on
// PreviousMatch and NextMatch, unless a scan is running, returning once its
// scan ends; hands any other `msg` to every Pass, cancelling any scan running,
// and returns their errors joined. A SearchFor starting a Search Pins the
// Window where it is.
func (e *Executor) Update(msg any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch msg.(type) {
	case FindCurrentLine:
		e.scan(e.findCurrentLine)
		return nil
	case PreviousMatch:
		e.scan(e.stepOlder)
		return nil
	case NextMatch:
		e.scan(e.stepNewer)
		return nil
	}

	e.gen++
	e.scanning = false
	var errs []error
	for _, pass := range e.passes {
		errs = append(errs, pass.Update(msg))
	}
	errs = append(errs, e.search.Update(msg))

	if _, ok := msg.(SearchFor); ok && e.search.searching() {
		e.pinWhereItIs()
	}
	return errors.Join(errs...)
}

// scan - runs `step` unless a scan is already running. Callers hold mu.
func (e *Executor) scan(step func(gen int)) {
	if e.scanning {
		return
	}
	e.scanning = true
	gen := e.gen
	step(gen)
	if gen == e.gen {
		e.scanning = false
	}
}

// read - the lines in [from, to), read with mu released so the UI can render
// and cancel; `false` if a message reached the Passes meanwhile.
func (e *Executor) read(gen, from, to int) ([]reader.Line, bool) {
	e.mu.Unlock()
	lines := e.Stream.Read(from, to)
	e.mu.Lock()
	return lines, gen == e.gen
}

// pinWhereItIs - Pins the Window where it is; no Line in it, nor above it,
// has been found to have a Match yet.
func (e *Executor) pinWhereItIs() {
	window := e.Stream.Lines()
	e.none = -1
	if len(window) == 0 {
		return
	}
	e.Stream.PinCentredOn(window[(len(window)-1)/2].Number)
	e.none = window[len(window)-1].Number
}

// findCurrentLine - while a Search has no Current Line, makes the newest Line
// with a Match at or above the Window's bottom Line when it started the
// Current Line, and centres the Window on it.
func (e *Executor) findCurrentLine(gen int) {
	if !e.search.searching() || e.search.current >= 0 {
		return
	}
	if number, ok := e.findOlder(gen, e.none+1); ok {
		e.search.current = number
		e.Stream.PinCentredOn(number)
	}
}

// stepOlder - makes the previous Line with a Match the Current Line and
// centres the Window on it; does nothing at the oldest one.
func (e *Executor) stepOlder(gen int) {
	e.dropUnreadCurrentLine()
	if e.search.current < 0 {
		return
	}
	if number, ok := e.findOlder(gen, e.search.current); ok {
		e.search.current = number
		e.Stream.PinCentredOn(number)
	}
}

// stepNewer - makes the next Line with a Match the Current Line and centres
// the Window on it; does nothing at the newest one.
func (e *Executor) stepNewer(gen int) {
	if !e.search.searching() {
		return
	}
	e.dropUnreadCurrentLine()
	after := e.search.current
	if after < 0 {
		after = e.none
	}
	if number, ok := e.findNewer(gen, after); ok {
		e.search.current = number
		e.Stream.PinCentredOn(number)
	}
}

// dropUnreadCurrentLine - drops the Current Line, and every Line with a Match
// found so far, if it can no longer be read (the Log File failed to read);
// stepping newer then starts from the oldest line that can.
func (e *Executor) dropUnreadCurrentLine() {
	current := e.search.current
	if current < 0 || current >= e.Stream.First() {
		return
	}
	e.search.current = -1
	e.search.seen = seen{}
	e.none = current
}

// findNewer - the Line Number of the oldest Line with a Match newer than line
// `after`, up to the newest line: one the Search remembers, or else one found
// by reading the History a batch at a time. `false` if there is none, or the
// scan was cancelled.
func (e *Executor) findNewer(gen, after int) (int, bool) {
	seen := e.search.seenFrom(after + 1)
	if i, _ := slices.BinarySearch(seen.lines, after+1); i < len(seen.lines) {
		return seen.lines[i], true
	}

	for seen.to < e.Stream.Total() {
		lines, ok := e.read(gen, seen.to, seen.to+scanBatch)
		if !ok || len(lines) == 0 {
			return 0, false
		}
		found := withMatch(e.run(lines))
		seen.lines = append(seen.lines, found...)
		seen.to = lines[len(lines)-1].Number + 1
		if len(found) > 0 {
			return found[0], true
		}
	}
	return 0, false
}

// findOlder - the Line Number of the newest Line with a Match older than line
// `before`: one the Search remembers, or else one found by reading the History
// a batch at a time. `false` if there is none, or the scan was cancelled.
func (e *Executor) findOlder(gen, before int) (int, bool) {
	seen := e.search.seenFrom(before)
	if i, _ := slices.BinarySearch(seen.lines, before); i > 0 {
		return seen.lines[i-1], true
	}

	for seen.from > 0 {
		from := max(seen.from-scanBatch, 0)
		lines, ok := e.read(gen, from, seen.from)
		if !ok {
			return 0, false
		}
		found := withMatch(e.run(lines))
		seen.lines = append(found, seen.lines...)
		seen.from = from
		if len(found) > 0 {
			return found[len(found)-1], true
		}
	}
	return 0, false
}

// withMatch - the Line Numbers of the Lines in `lines` with a Match.
func withMatch(lines []reader.Line) []int {
	var numbers []int
	for _, line := range lines {
		if len(line.Matches) > 0 {
			numbers = append(numbers, line.Number)
		}
	}
	return numbers
}

// Lines - the lines in the Window, after every Pass.
func (e *Executor) Lines() []reader.Line {
	lines := e.Stream.Lines()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.run(lines)
}

// run - copies of `lines` after every Pass, the Search last. Callers hold mu.
func (e *Executor) run(lines []reader.Line) []reader.Line {
	for i := range lines {
		lines[i].Shown = lines[i].Text
	}
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
