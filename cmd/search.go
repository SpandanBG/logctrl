package cmd

import (
	"regexp"
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// SearchFor - the message starting a Search for Regex; an empty Regex ends it.
type SearchFor struct {
	Regex string
}

// Search - the Pass marking where the Search regex matches each Line, and
// highlighting each Match.
type Search struct {
	regex     *regexp.Regexp // nil while idle
	current   int            // Line Number of the Current Line; -1 if none
	seen      seen           // the Lines with a Match found so far
	highlight lipgloss.Style // how each Match is shown
	bold      lipgloss.Style // how the rest of the Current Line is shown
}

// NewSearch - an idle Search, whose Matches are styled by `renderer`.
func NewSearch(renderer *lipgloss.Renderer) *Search {
	return &Search{
		current: -1,
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
		bold: renderer.NewStyle().Bold(true),
	}
}

// Update - on SearchFor, searches for its regex, or goes idle if it is empty.
// A regex that does not compile leaves the Search idle and is returned as the
// error. Any other message only forgets the Lines with a Match found so far
// (see docs/adr/0007).
func (s *Search) Update(msg any) error {
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
	return nil
}

// Run - sets the Matches of each line in its shown text, then highlights them
// there; empty matches are dropped. Leaves the lines as they are while idle.
func (s *Search) Run(lines []reader.Line) []reader.Line {
	if s.regex == nil {
		return lines
	}

	for i := range lines {
		lines[i].Matches = nil
		for _, at := range s.regex.FindAllStringIndex(lines[i].Shown, -1) {
			if at[0] < at[1] {
				lines[i].Matches = append(lines[i].Matches, reader.Match{Start: at[0], End: at[1]})
			}
		}
		lines[i].Shown = s.highlighted(lines[i])
	}
	return lines
}

// highlighted - the shown text of `line` with each of its Matches
// highlighted, all of it bold if it is the Current Line.
func (s *Search) highlighted(line reader.Line) string {
	plain := func(text string) string { return text }
	highlight := s.highlight.Render
	if line.Number == s.current {
		plain = func(text string) string {
			if text == "" {
				return ""
			}
			return s.bold.Render(text)
		}
		highlight = s.highlight.Bold(true).Render
	}

	var text strings.Builder
	from := 0
	for _, match := range line.Matches {
		text.WriteString(plain(line.Shown[from:match.Start]))
		text.WriteString(highlight(line.Shown[match.Start:match.End]))
		from = match.End
	}
	text.WriteString(plain(line.Shown[from:]))
	return text.String()
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
