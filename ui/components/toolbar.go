package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const helpText = ui.Grey_Color + "Quick Help:\t\t" +
	ui.Magenta_Color + "q" + ui.Black_Color + ":Quit" +
	ui.Reset_Color

var (
	toolbarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("43")).
		Foreground(lipgloss.Color("0"))
)

type toolbar struct {
	width    ui.SizeI
	height   ui.SizeI
	rendered string
}

func NewToolbar(width, height ui.SizeI) tea.Model {
	return toolbar{
		width:  width,
		height: height,
	}
}

func (t toolbar) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
	)
}

func (t toolbar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return t.updateView(msg)
	}

	return t, nil
}

func (t toolbar) View() string {
	return t.rendered
}

// ------------------------- Private
func (t toolbar) updateView(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	size = ui.ModifySize(size, t.width, t.height)

	t.rendered = toolbarStyle.
		Width(size.Width).
		Height(size.Height).
		Render(helpText)

	return t, nil
}
