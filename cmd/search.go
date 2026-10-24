package cmd

import (
	"regexp"
	"slices"
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// SearchFor - the message starting a Search for Regex; an empty Regex ends it.
type SearchFor struct {
	Regex string
}

// FindCurrentLine - the message finding the Current Line once a SearchFor has
// started a Search: the newest Line with a Match at or above the Window's
// bottom Line. The UI sends it in the background, as it may scan the History
// (see docs/adr/0007).
type FindCurrentLine struct{}

// PreviousMatch - the message stepping from the Current Line to the previous
// (older) Line with a Match.
type PreviousMatch struct{}

// NextMatch - the message stepping from the Current Line to the next (newer)
// Line with a Match.
type NextMatch struct{}

// scanBatch - Lines read from the History at a time while looking for a Line
// with a Match outside the Window: a Chunk's worth (see docs/adr/0007).
const scanBatch = 1_000

// history - what the Search may see of the History, and do to the Window
// (see docs/adr/0008).
type history interface {
	Window() []int                   // Line Numbers of the Lines in the Window
	Total() int                      // lines received so far
	First() int                      // Line Number of the oldest line that can be read
	Read(from, to int) []reader.Line // lines in [from, to) as the other Passes leave them
	PinCentredOn(number int)         // Pins the Window centred on line `number`
}

// Search - the Pass marking where the Search regex matches each Line, and
// highlighting each Match. It steps between Lines with a Match, scanning the
// History (see docs/adr/0007).
type Search struct {
	regex     *regexp.Regexp // nil while idle
	current   int            // Line Number of the Current Line; -1 if none
	seen      seen           // the Lines with a Match found so far
	none      int            // while there is no Current Line, the newest Line known to have no Match
	gen       int            // counts the messages that are not steps; a scan from an older one is cancelled
	scanning  bool           // a scan for message `gen` is running
	history   history
	highlight lipgloss.Style // how each Match is shown
}

// NewSearch - an idle Search, whose Matches are styled by `renderer`.
func NewSearch(renderer *lipgloss.Renderer) *Search {
	return &Search{
		current: -1,
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
	}
}

// attach - gives the Search the History to step through.
func (s *Search) attach(h history) {
	s.history = h
}

// Update - finds the Current Line on FindCurrentLine and steps on
// PreviousMatch and NextMatch, unless a scan is running, returning once its
// scan ends. Any other message cancels any scan running and forgets the Lines
// with a Match found so far (see docs/adr/0007). On SearchFor, searches for
// its regex, Pinning the Window where it is, or goes idle if it is empty; a
// regex that does not compile leaves the Search idle and is returned as the
// error.
func (s *Search) Update(msg any) error {
	switch msg.(type) {
	case FindCurrentLine:
		s.scan(s.findCurrentLine)
		return nil
	case PreviousMatch:
		s.scan(s.stepOlder)
		return nil
	case NextMatch:
		s.scan(s.stepNewer)
		return nil
	}

	s.gen++
	s.scanning = false
	s.seen = seen{} // any message may change the shown text matched
	searchFor, ok := msg.(SearchFor)
	if !ok {
		return nil
	}

	s.regex = nil
	s.current = -1
	if searchFor.Regex == "" {
		return nil
	}

	compiled, err := regexp.Compile(searchFor.Regex)
	if err != nil {
		return err
	}
	s.regex = compiled
	s.pinWhereItIs()
	return nil
}

// Run - sets the Matches of each line in its shown text, then highlights them
// there; empty matches are dropped. Marks the Current Line for the Gutter.
// Leaves the lines as they are while idle.
func (s *Search) Run(lines []reader.Line) []reader.Line {
	if s.regex == nil {
		return lines
	}

	for i := range lines {
		lines[i].Matches = s.matches(lines[i].Shown)
		lines[i].GutterContext.Search.Current = lines[i].Number == s.current
		lines[i].Shown = s.highlighted(lines[i])
	}
	return lines
}

// matches - where the regex matches `shown`; empty matches are dropped.
func (s *Search) matches(shown string) []reader.Match {
	var matches []reader.Match
	for _, at := range s.regex.FindAllStringIndex(shown, -1) {
		if at[0] < at[1] {
			matches = append(matches, reader.Match{Start: at[0], End: at[1]})
		}
	}
	return matches
}

// highlighted - the shown text of `line` with each of its Matches
// highlighted.
func (s *Search) highlighted(line reader.Line) string {
	var text strings.Builder
	from := 0
	for _, match := range line.Matches {
		text.WriteString(line.Shown[from:match.Start])
		text.WriteString(s.highlight.Render(line.Shown[match.Start:match.End]))
		from = match.End
	}
	text.WriteString(line.Shown[from:])
	return text.String()
}

// scan - runs `step` unless a scan is already running.
func (s *Search) scan(step func(gen int)) {
	if s.scanning {
		return
	}
	s.scanning = true
	gen := s.gen
	step(gen)
	if gen == s.gen {
		s.scanning = false
	}
}

// read - the lines in [from, to); `false` if a message other than a step
// reached the Search meanwhile.
func (s *Search) read(gen, from, to int) ([]reader.Line, bool) {
	lines := s.history.Read(from, to)
	return lines, gen == s.gen
}

// pinWhereItIs - Pins the Window where it is; no Line in it, nor above it,
// has been found to have a Match yet.
func (s *Search) pinWhereItIs() {
	window := s.history.Window()
	s.none = -1
	if len(window) == 0 {
		return
	}
	s.history.PinCentredOn(window[(len(window)-1)/2])
	s.none = window[len(window)-1]
}

// findCurrentLine - while a Search has no Current Line, makes the newest Line
// with a Match at or above the Window's bottom Line when it started the
// Current Line, and centres the Window on it.
func (s *Search) findCurrentLine(gen int) {
	if !s.searching() || s.current >= 0 {
		return
	}
	if number, ok := s.findOlder(gen, s.none+1); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// stepOlder - makes the previous Line with a Match the Current Line and
// centres the Window on it; does nothing at the oldest one.
func (s *Search) stepOlder(gen int) {
	s.dropUnreadCurrentLine()
	if s.current < 0 {
		return
	}
	if number, ok := s.findOlder(gen, s.current); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// stepNewer - makes the next Line with a Match the Current Line and centres
// the Window on it; does nothing at the newest one.
func (s *Search) stepNewer(gen int) {
	if !s.searching() {
		return
	}
	s.dropUnreadCurrentLine()
	after := s.current
	if after < 0 {
		after = s.none
	}
	if number, ok := s.findNewer(gen, after); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// dropUnreadCurrentLine - drops the Current Line, and every Line with a Match
// found so far, if it can no longer be read (the Log File failed to read);
// stepping newer then starts from the oldest line that can.
func (s *Search) dropUnreadCurrentLine() {
	current := s.current
	if current < 0 || current >= s.history.First() {
		return
	}
	s.current = -1
	s.seen = seen{}
	s.none = current
}

// findNewer - the Line Number of the oldest Line with a Match newer than line
// `after`, up to the newest line: one the Search remembers, or else one found
// by reading the History a batch at a time. `false` if there is none, or the
// scan was cancelled.
func (s *Search) findNewer(gen, after int) (int, bool) {
	seen := s.seenFrom(after + 1)
	if i, _ := slices.BinarySearch(seen.lines, after+1); i < len(seen.lines) {
		return seen.lines[i], true
	}

	for seen.to < s.history.Total() {
		lines, ok := s.read(gen, seen.to, seen.to+scanBatch)
		if !ok || len(lines) == 0 {
			return 0, false
		}
		found := s.withMatch(lines)
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
func (s *Search) findOlder(gen, before int) (int, bool) {
	seen := s.seenFrom(before)
	if i, _ := slices.BinarySearch(seen.lines, before); i > 0 {
		return seen.lines[i-1], true
	}

	for seen.from > 0 {
		from := max(seen.from-scanBatch, 0)
		lines, ok := s.read(gen, from, seen.from)
		if !ok {
			return 0, false
		}
		found := s.withMatch(lines)
		seen.lines = append(found, seen.lines...)
		seen.from = from
		if len(found) > 0 {
			return found[len(found)-1], true
		}
	}
	return 0, false
}

// withMatch - the Line Numbers of the Lines in `lines` with a Match.
func (s *Search) withMatch(lines []reader.Line) []int {
	var numbers []int
	for _, line := range lines {
		if len(s.matches(line.Shown)) > 0 {
			numbers = append(numbers, line.Number)
		}
	}
	return numbers
}

// seen - the Lines with a Match found so far for the regex: every Line Number
// in [from, to) has been scanned, and `lines` holds those with a Match, oldest
// first.
type seen struct {
	from, to int
	lines    []int
}

// seenFrom - what has been found so far, if it reaches line `number`; else
// forgets it and starts again there.
func (s *Search) seenFrom(number int) *seen {
	if number < s.seen.from || number > s.seen.to {
		s.seen = seen{from: number, to: number}
	}
	return &s.seen
}

// searching - whether a regex is being searched for.
func (s *Search) searching() bool {
	return s.regex != nil
}
