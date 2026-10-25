package cmd

import (
	"regexp"

	"github.com/SpandanBG/logctrl/reader"
)

// FilterFor - the message starting a Filter for Regex; an empty Regex ends it.
type FilterFor struct {
	Regex string
}

// filter - the Pass hiding each Line whose shown text the Filter regex does
// not match.
type filter struct {
	regex *regexp.Regexp // nil while idle
}

// newFilter - an idle Filter.
func newFilter() *filter {
	return &filter{}
}

// Update - on FilterFor, filters for its regex, or goes idle if it is empty; a
// regex that does not compile leaves the Filter idle and is returned as the
// error.
func (f *filter) Update(msg any) error {
	filterFor, ok := msg.(FilterFor)
	if !ok {
		return nil
	}
	f.regex = nil
	if filterFor.Regex == "" {
		return nil
	}
	compiled, err := regexp.Compile(filterFor.Regex)
	if err != nil {
		return err
	}
	f.regex = compiled
	return nil
}

// Run - hides each line whose shown text the regex does not match. Leaves the
// lines as they are while idle.
func (f *filter) Run(lines []reader.Line) []reader.Line {
	if f.regex == nil {
		return lines
	}
	for i := range lines {
		lines[i].Filter.Hide = !f.regex.MatchString(lines[i].Shown)
	}
	return lines
}
