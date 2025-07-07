package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// ----- Public tea.Msg
type TeaPromptToggle struct {
	BringFocus bool
}

type prompt struct {
	width    ui.SizeI
	height   ui.SizeI
	size     tea.WindowSizeMsg
	rendered string
	view     textarea.Model
	focused  bool
}

func NewPrompt(width, height ui.SizeI) tea.Model {
	return &prompt{
		width:   width,
		height:  height,
		view:    textarea.New(),
		focused: false,
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
	case TeaPromptToggle:
		return p.setFocus(msg)
	}

	var cmd tea.Cmd
	p.view, cmd = p.view.Update(msg)
	return p, cmd
}

func (p prompt) View() string {
	return p.view.View()
}

// ------------------------- Private
func (p prompt) updateView(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	p.size = ui.ModifySize(size, p.width, p.height)
	p.view.SetWidth(p.size.Width)
	p.view.SetHeight(p.size.Height)
	return p, nil
}

func (p prompt) setFocus(data TeaPromptToggle) (tea.Model, tea.Cmd) {
	p.focused = data.BringFocus

	if p.focused {
		return p, p.view.Focus()
	}

	p.view.Blur()
	return p, nil
}
