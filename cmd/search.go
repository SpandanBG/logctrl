package cmd

import (
	"regexp"

	"github.com/SpandanBG/logctrl/reader"
)

// Search - the Pass marking where the Search regex matches each Line.
type Search struct {
	regex *regexp.Regexp // nil while no Search is running
}

// NewSearch - a Search with no regex, which marks nothing.
func NewSearch() *Search {
	return &Search{}
}

// Set - starts a Search for `regex`, or ends it if `regex` is empty. A regex
// that does not compile ends the Search and is returned as the error.
func (s *Search) Set(regex string) error {
	s.regex = nil
	if regex == "" {
		return nil
	}

	compiled, err := regexp.Compile(regex)
	if err != nil {
		return err
	}
	s.regex = compiled
	return nil
}

// Run - sets the Matches of each line; empty matches are dropped.
func (s *Search) Run(lines []reader.Line) []reader.Line {
	if s.regex == nil {
		return lines
	}

	for i := range lines {
		lines[i].Matches = nil
		for _, at := range s.regex.FindAllStringIndex(lines[i].Text, -1) {
			if at[0] < at[1] {
				lines[i].Matches = append(lines[i].Matches, reader.Match{Start: at[0], End: at[1]})
			}
		}
	}
	return lines
}
