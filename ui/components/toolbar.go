package components

import (
	"github.com/SpandanBG/logctrl/config"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpText - the keys that open the Help, Scroll down/up, open the Search Bar
// and step to the next/previous Line with a Match.
func helpText(keymap []config.Entry) string {
	keyOf := map[config.Action]string{}
	for _, entry := range keymap {
		keyOf[entry.Action] = entry.Key
	}

	return ui.Grey_Color + "Quick Help:\t\t" +
		ui.Magenta_Color + keyOf[config.OpenHelp] + ui.Black_Color + ":Help  " +
		ui.Magenta_Color + keyOf[config.ScrollDown] + "/" + keyOf[config.ScrollUp] +
		ui.Black_Color + ":Scroll  " +
		ui.Magenta_Color + keyOf[config.OpenSearch] + ui.Black_Color + ":Search  " +
		ui.Magenta_Color + keyOf[config.NextMatch] + "/" + keyOf[config.PreviousMatch] +
		ui.Black_Color + ":Match" +
		ui.Reset_Color
}

var (
	toolbarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("43")).
		Foreground(lipgloss.Color("0"))
)

type toolbar struct {
	width    ui.SizeI
	height   ui.SizeI
	help     string
	rendered string
}

func NewToolbar(width, height ui.SizeI, keymap []config.Entry) tea.Model {
	return toolbar{
		width:  width,
		height: height,
		help:   helpText(keymap),
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
		Render(t.help)

	return t, nil
}
