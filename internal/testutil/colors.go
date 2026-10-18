package testutil

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Colors - a renderer that always outputs ANSI colors, as a terminal would,
// whatever the test runs in.
func Colors() *lipgloss.Renderer {
	renderer := lipgloss.NewRenderer(io.Discard)
	renderer.SetColorProfile(termenv.ANSI)
	return renderer
}
