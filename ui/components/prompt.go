package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var promptStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("56"))

type prompt struct {
	width    ui.SizeI
	height   ui.SizeI
	rendered string
}

func NewPrompt(width, height ui.SizeI) tea.Model {
	return &prompt{
		width:  width,
		height: height,
	}
}

func (p prompt) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
	)
}

func (p prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return p.updateView(msg)
	}

	return p, nil
}

func (p prompt) View() string {
	return p.rendered
}

// ------------------------- Private
func (p prompt) updateView(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	size = ui.ModifySize(size, p.width, p.height)

	p.rendered = promptStyle.
		Width(size.Width).
		Height(size.Height).
		Render("") // TODO: replace with input field

	return p, nil
}
