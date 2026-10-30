package reader

// GutterWidth - the cells in a Line's Gutter.
const GutterWidth = 2

// Line - one entry of the Log Feed, as held in the LiveBuffer or a Chunk.
type Line struct {
	Number        int               // its Line Number
	Text          string            // without its terminator
	Shown         string            // Text as formatted by the Passes, for display; empty as held
	Matches       []Match           // where the Search regex matches Shown; nil if nowhere
	Filter        FilterContext     // what the Filter works out about the Line; zero as held
	GutterContext GutterContext     // what the Passes leave for the Gutter Pass; zero as held
	Gutter        [GutterWidth]rune // its Gutter cells, set by the Gutter Pass; 0 is an empty cell
}

// Match - a section of a Line's shown text that the Search regex matches, as
// byte offsets [Start, End).
type Match struct {
	Start int // byte offset of its first byte
	End   int // byte offset just after its last byte
}

// FilterContext - what the Filter works out about a Line.
type FilterContext struct {
	Hide bool // the Line is hidden from the Log View
}

// GutterContext - what the Passes have worked out about a Line that the
// Gutter Pass may mark in its Gutter, by the Pass that worked it out (see
// docs/adr/0009).
type GutterContext struct {
	Search SearchContext // what the Search leaves
}

// SearchContext - what the Search leaves for the Gutter Pass.
type SearchContext struct {
	Current bool // the Line is the Current Line
}
