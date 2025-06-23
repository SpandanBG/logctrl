package utils

import tea "github.com/charmbracelet/bubbletea"

func ModifySize(size tea.WindowSizeMsg, w, h SizeI) tea.WindowSizeMsg {
	size.Width = updateSize(size.Width, w)
	size.Height = updateSize(size.Height, h)
	return size
}

func updateSize(x int, xm SizeI) int {
	switch xm.Type() {
	case Ratio:
		return int(float32(xm.(SizeRatio)) * float32(x))
	case Fixed:
		return int(xm.(SizeFixed))
	case Modifier:
		return x + int(xm.(SizeModifier))
	default:
		return x
	}
}
