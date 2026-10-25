package cmd

import "github.com/SpandanBG/logctrl/reader"

// gutter - the Pass run after every other, setting the Gutter cells of each
// Line from the context the other Passes leave on it (see docs/adr/0009).
type gutter struct{}

// Update - ignores every message: the Gutter only reads each Line.
func (g *gutter) Update(msg any) error {
	return nil
}

// Run - sets the Gutter cells of each line: `>` in cell 1 beside the Current
// Line.
func (g *gutter) Run(lines []reader.Line) []reader.Line {
	for i := range lines {
		if lines[i].GutterContext.Search.Current {
			lines[i].Gutter[1] = '>'
		}
	}
	return lines
}

// ------------------------- Private

// newGutter - the Gutter Pass.
func newGutter() *gutter {
	return &gutter{}
}
