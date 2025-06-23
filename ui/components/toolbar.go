package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	helpText = ui.Grey_Color + "Quick Help:\t\t" +
		ui.Magenta_Color + "q" + ui.Black_Color + ":Quit" +
		ui.Reset_Color
)

var (
	toolbarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("43")).
		Foreground(lipgloss.Color("0"))
)

type toolbar struct {
	width  ui.SizeI
	height ui.SizeI
	view   viewport.Model
	ready  bool
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
		return t.updateViewSize(msg)
	}

	return t, nil
}

func (t toolbar) View() string {
	return toolbarStyle.Render(t.view.View())
}

// ------------------------- Private
func (t toolbar) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	size = ui.ModifySize(size, t.width, t.height)

	w := size.Width - toolbarStyle.GetHorizontalFrameSize()
	h := size.Height - toolbarStyle.GetVerticalFrameSize()

	if t.ready {
		t.view.Width = w
		t.view.Height = h
	} else {
		t.view = viewport.New(w, h)
		t.ready = true

		t.view.SetContent(helpText)
	}

	return t, nil
}
