package cmd

import "github.com/SpandanBG/logctrl/reader"

// Gutter - the Pass run after every other, setting the Gutter cells of each
// Line from the context the other Passes leave on it (see docs/adr/0009).
type Gutter struct{}

// NewGutter - the Gutter Pass.
func NewGutter() *Gutter {
	return &Gutter{}
}

// Update - ignores every message: the Gutter only reads each Line.
func (g *Gutter) Update(msg any) error {
	return nil
}

// Run - sets the Gutter cells of each line: `>` in cell 1 beside the Current
// Line.
func (g *Gutter) Run(lines []reader.Line) []reader.Line {
	for i := range lines {
		if lines[i].GutterContext.Search.Current {
			lines[i].Gutter[1] = '>'
		}
	}
	return lines
}
