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
	highlight lipgloss.Style // how each Match is shown
}

// NewSearch - an idle Search, whose Matches are styled by `renderer`.
func NewSearch(renderer *lipgloss.Renderer) *Search {
	return &Search{
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
	}
}

// Update - on SearchFor, searches for its regex, or goes idle if it is empty.
// A regex that does not compile leaves the Search idle and is returned as the
// error. Any other message is ignored.
func (s *Search) Update(msg any) error {
	searchFor, ok := msg.(SearchFor)
	if !ok {
		return nil
	}

	s.regex = nil
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

// highlighted - the shown text of `line` with each of its Matches highlighted.
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
