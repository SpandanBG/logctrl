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
	Regex string // the regex to search for; "" ends the Search
}

// PreviousMatch - the message stepping from the Current Line to the previous
// (older) Line with a Match.
type PreviousMatch struct{}

// NextMatch - the message stepping from the Current Line to the next (newer)
// Line with a Match.
type NextMatch struct{}

// Follow - the message the UI sends as the user returns the Window to
// Following, dropping the Current Line.
type Follow struct{}

// scanBatch - Lines read from the History at a time while looking for a Line
// with a Match outside the Window: a Chunk's worth (see docs/adr/0007).
const scanBatch = 1_000

// history - what the Search may see of the History, and do to the Window
// (see docs/adr/0008).
type history interface {
	Window() []int                   // Line Numbers of the Lines in the Window the Filter does not hide
	Total() int                      // lines received so far
	First() int                      // Line Number of the oldest line that can be read
	Read(from, to int) []reader.Line // lines in [from, to) as the other Passes leave them
	PinCentredOn(number int)         // Pins the Window centred on line `number`
	Follow()                         // makes the Window Following
}

// search - the Pass marking where the Search regex matches each Line, and
// highlighting each Match. It steps between Lines with a Match, scanning the
// History (see docs/adr/0007).
type search struct {
	regex     *regexp.Regexp // nil while idle
	current   int            // Line Number of the Current Line; -1 if none
	seen      seen           // the Lines with a Match found so far
	dropped   int            // a Current Line dropped once it could not be read; -1 if none
	gen       int            // counts the messages that are not steps; a scan from an older one is cancelled
	scanning  bool           // a scan for message `gen` is running
	history   history        // the History to step through, as the Passes before the Search leave it
	highlight lipgloss.Style // how each Match is shown
}

// Update - steps on PreviousMatch and NextMatch, finding the Current Line
// first if there is none, unless a scan is running, returning once its scan
// ends. Resize changes nothing, so it cancels no step. Any other message
// cancels any scan running and forgets the Lines with a Match found so far
// (see docs/adr/0007); Follow also drops the Current Line, and SearchFor
// searches for its regex, unless it is the one already searched for, which
// changes nothing.
func (s *search) Update(msg any) error {
	switch msg := msg.(type) {
	case PreviousMatch:
		s.scan(s.stepOlder)
		return nil
	case NextMatch:
		s.scan(s.stepNewer)
		return nil
	case Resize:
		return nil
	case SearchFor:
		if s.searching() && s.regex.String() == msg.Regex {
			return nil
		}
		s.cancel()
		return s.searchFor(msg.Regex)
	case Follow:
		s.cancel()
		s.current = -1
		s.dropped = -1
		return nil
	}
	s.cancel()
	return nil
}

// Run - sets the Matches of each line in its shown text, then highlights them
// there; empty matches are dropped. Marks the Current Line for the Gutter.
// Leaves the lines as they are while idle.
func (s *search) Run(lines []reader.Line) []reader.Line {
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

// seen - the Lines with a Match found so far for the regex: every Line Number
// in [from, to) has been scanned, and `lines` holds those with a Match, oldest
// first.
type seen struct {
	from, to int   // the Line Numbers scanned: [from, to)
	lines    []int // the Line Numbers of those with a Match, oldest first
}

// ------------------------- Private

// newSearch - an idle Search, whose Matches are styled by `renderer`.
func newSearch(renderer *lipgloss.Renderer) *search {
	return &search{
		current: -1,
		dropped: -1,
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
	}
}

// cancel - cancels any scan running and forgets the Lines with a Match found
// so far, as any message may change the shown text matched.
func (s *search) cancel() {
	s.gen++
	s.scanning = false
	s.seen = seen{}
}

// searchFor - searches for `regex`, dropping the Current Line, and makes the
// Window Following; goes idle, keeping the Window's mode, if `regex` is empty.
// A regex that does not compile leaves the Search idle, still Following, and
// is returned as the error.
func (s *search) searchFor(regex string) error {
	s.regex = nil
	s.current = -1
	s.dropped = -1
	if regex == "" {
		return nil
	}

	s.history.Follow()
	compiled, err := regexp.Compile(regex)
	if err != nil {
		return err
	}
	s.regex = compiled
	return nil
}

// attach - gives the Search the History to step through.
func (s *search) attach(h history) {
	s.history = h
}

// matches - where the regex matches `shown`; empty matches are dropped.
func (s *search) matches(shown string) []reader.Match {
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
func (s *search) highlighted(line reader.Line) string {
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
func (s *search) scan(step func(gen int)) {
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
func (s *search) read(gen, from, to int) ([]reader.Line, bool) {
	lines := s.history.Read(from, to)
	return lines, gen == s.gen
}

// findCurrentLine - while a Search has no Current Line, makes the newest Line
// with a Match at or above the Window's bottom Line the Current Line, and
// centres the Window on it.
func (s *search) findCurrentLine(gen int) {
	if !s.searching() || s.current >= 0 {
		return
	}
	window := s.history.Window()
	if len(window) == 0 {
		return
	}
	if number, ok := s.findOlder(gen, window[len(window)-1]+1); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// stepOlder - makes the previous Line with a Match the Current Line and
// centres the Window on it; does nothing at the oldest one. Finds the Current
// Line if there is none.
func (s *search) stepOlder(gen int) {
	if s.dropUnreadCurrentLine() || s.current < 0 && s.dropped >= 0 {
		return
	}
	if s.current < 0 {
		s.findCurrentLine(gen)
		return
	}
	if number, ok := s.findOlder(gen, s.current); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// stepNewer - makes the next Line with a Match the Current Line and centres
// the Window on it; does nothing at the newest one. Finds the Current Line if
// there is none.
func (s *search) stepNewer(gen int) {
	if !s.searching() {
		return
	}
	s.dropUnreadCurrentLine()
	if s.current < 0 && s.dropped < 0 {
		s.findCurrentLine(gen)
		return
	}
	after := s.current
	if after < 0 {
		after = s.dropped
	}
	if number, ok := s.findNewer(gen, after); ok {
		s.current = number
		s.history.PinCentredOn(number)
	}
}

// dropUnreadCurrentLine - drops the Current Line, and every Line with a Match
// found so far, if it can no longer be read (the Log File failed to read);
// stepping newer then starts from the oldest line that can. `true` if it was
// dropped.
func (s *search) dropUnreadCurrentLine() bool {
	current := s.current
	if current < 0 || current >= s.history.First() {
		return false
	}
	s.current = -1
	s.seen = seen{}
	s.dropped = current
	return true
}

// findNewer - the Line Number of the oldest Line with a Match newer than line
// `after`, up to the newest line: one the Search remembers, or else one found
// by reading the History a batch at a time. `false` if there is none, or the
// scan was cancelled.
func (s *search) findNewer(gen, after int) (int, bool) {
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
func (s *search) findOlder(gen, before int) (int, bool) {
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

// withMatch - the Line Numbers of the Lines in `lines` with a Match that the
// Filter does not hide.
func (s *search) withMatch(lines []reader.Line) []int {
	var numbers []int
	for _, line := range lines {
		if !line.Filter.Hide && len(s.matches(line.Shown)) > 0 {
			numbers = append(numbers, line.Number)
		}
	}
	return numbers
}

// seenFrom - what has been found so far, if it reaches line `number`; else
// forgets it and starts again there.
func (s *search) seenFrom(number int) *seen {
	if number < s.seen.from || number > s.seen.to {
		s.seen = seen{from: number, to: number}
	}
	return &s.seen
}

// searching - whether a regex is being searched for.
func (s *search) searching() bool {
	return s.regex != nil
}
