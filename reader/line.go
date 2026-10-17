package reader

// Line - one entry of the Log Feed, as held in the LiveBuffer or a Chunk.
type Line struct {
	Text    string  // without its terminator
	Matches []Match // where the Search regex matches Text; nil if nowhere
}

// Match - a section of a Line's Text that the Search regex matches, as byte
// offsets [Start, End).
type Match struct {
	Start int
	End   int
}
