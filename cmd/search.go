package cmd

import (
	"regexp"
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/lipgloss"
)

// Search - the Pass marking where the Search regex matches each Line, and
// highlighting each Match.
type Search struct {
	regex     *regexp.Regexp
	highlight lipgloss.Style // how each Match is shown
}

// NewSearch - a Search for `regex`, whose Matches are styled by `renderer`. A
// regex that does not compile is returned as the error.
func NewSearch(renderer *lipgloss.Renderer, regex string) (*Search, error) {
	compiled, err := regexp.Compile(regex)
	if err != nil {
		return nil, err
	}

	return &Search{
		regex: compiled,
		highlight: renderer.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("3")),
	}, nil
}

// Run - sets the Matches of each line in its shown text, then highlights them
// there; empty matches are dropped.
func (s *Search) Run(lines []reader.Line) []reader.Line {
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
