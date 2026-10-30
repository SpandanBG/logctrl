package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// ----- Public tea.Msg

// TeaPromptToggle - focuses or blurs the Prompt.
type TeaPromptToggle struct {
	BringFocus bool // focus it if true, blur it if false
}

// prompt - the Prompt: a text area the user types a command in.
type prompt struct {
	width   ui.SizeI          // width modifier
	height  ui.SizeI          // height modifier
	size    tea.WindowSizeMsg // its size, the terminal's after the modifiers
	view    textarea.Model    // the text area
	focused bool              // whether typing goes to it
}

// NewPrompt - an unfocused Prompt sized by the `width` and `height` modifiers.
func NewPrompt(width, height ui.SizeI) tea.Model {
	return &prompt{
		width:   width,
		height:  height,
		view:    textarea.New(),
		focused: false,
	}
}

// Init - asks for the terminal's size.
func (p prompt) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
	)
}

// Update - resizes on a new terminal size, focuses or blurs on
// TeaPromptToggle, and hands any other message to the text area.
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

// View - the text area.
func (p prompt) View() string {
	return p.view.View()
}

// ------------------------- Private

// updateView - fits the text area to its share of the terminal `size`.
func (p prompt) updateView(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	p.size = ui.ModifySize(size, p.width, p.height)
	p.view.SetWidth(p.size.Width)
	p.view.SetHeight(p.size.Height)
	return p, nil
}

// setFocus - focuses the text area if `data` says so, else blurs it.
func (p prompt) setFocus(data TeaPromptToggle) (tea.Model, tea.Cmd) {
	p.focused = data.BringFocus

	if p.focused {
		return p, p.view.Focus()
	}

	p.view.Blur()
	return p, nil
}
