package reader

// Line - one entry of the Log Feed, as held in the LiveBuffer or a Chunk.
type Line struct {
	Text    string  // without its terminator
	Shown   string  // Text as formatted by the Passes, for display; empty as held
	Matches []Match // where the Search regex matches Shown; nil if nowhere
}

// Match - a section of a Line's shown text that the Search regex matches, as
// byte offsets [Start, End).
type Match struct {
	Start int
	End   int
}
