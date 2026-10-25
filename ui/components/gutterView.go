package components

import (
	"strings"

	"github.com/SpandanBG/logctrl/headless"
	"github.com/charmbracelet/lipgloss"
)

// gutterView - the Gutter down the left of the Log View: each Line's Gutter
// cells beside it, as tall as the Log View.
type gutterView struct {
	cells  []string // the Gutter cells of each Line in the Window, top first
	height int      // rows, as tall as the Log View
	style  lipgloss.Style
}

func (g gutterView) View() string {
	blank := strings.Repeat(" ", headless.GutterWidth)
	rows := make([]string, g.height)
	for i := range rows {
		rows[i] = blank
		if i < len(g.cells) {
			rows[i] = g.cells[i]
		}
	}
	return g.style.Render(strings.Join(rows, "\n"))
}

// ------------------------- Private

// newGutterView - an empty Gutter, styled by `renderer`.
func newGutterView(renderer *lipgloss.Renderer) gutterView {
	return gutterView{style: renderer.NewStyle().Background(lipgloss.Color("236"))}
}

// setCells - shows `cells`, the Gutter cells of each Line joined by newlines.
func (g gutterView) setCells(cells string) gutterView {
	g.cells = nil
	if cells != "" {
		g.cells = strings.Split(cells, "\n")
	}
	return g
}
