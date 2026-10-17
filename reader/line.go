package reader

// Line - one entry of the Log Feed, as held in the LiveBuffer or a Chunk.
type Line struct {
	Text string // without its terminator
}
