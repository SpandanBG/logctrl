package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// readOnlyStyle - the regex once entered: dimmed, with no cursor.
	readOnlyStyle = lipgloss.NewStyle().Faint(true)
)

// ----- Public tea.Msg

// TeaSearchEdit - makes the Search Bar editable, keeping its regex.
type TeaSearchEdit struct{}

// TeaSearchClose - empties the Search Bar, ready for the next Search.
type TeaSearchClose struct{}

// TeaSearchCancel - drops what was typed since the regex was last entered,
// making the Search Bar read-only again.
type TeaSearchCancel struct{}

// TeaSearchEntered - the user entered `Regex`; the Search Bar is read-only.
type TeaSearchEntered struct {
	Regex string
}

// TeaSearchFailed - the regex last entered cannot be searched for, because of
// `Err`.
type TeaSearchFailed struct {
	Err error
}

type searchBar struct {
	width      ui.SizeI
	view       textinput.Model
	entered    string         // the regex last entered
	failed     error          // why the regex last entered cannot be searched for, if so
	errorStyle lipgloss.Style // how `failed` is shown
}

// NewSearchBar - the one-line Search Bar, `width` wide, styled by `renderer`.
func NewSearchBar(width ui.SizeI, renderer *lipgloss.Renderer) tea.Model {
	view := textinput.New()
	view.Prompt = "/"
	return searchBar{
		width:      width,
		view:       view,
		errorStyle: renderer.NewStyle().Foreground(lipgloss.Color("1")),
	}
}

func (s searchBar) Init() tea.Cmd {
	return tea.WindowSize()
}

func (s searchBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.view.Width = ui.ModifySize(msg, s.width, ui.SizeFixed(1)).Width - len(s.view.Prompt) - 1
		return s, nil
	case TeaSearchEdit:
		s.view.PromptStyle = lipgloss.NewStyle()
		s.view.TextStyle = lipgloss.NewStyle()
		return s, s.view.Focus()
	case TeaSearchFailed:
		s.failed = msg.Err
		return s, nil
	case TeaSearchClose:
		s = s.readOnly()
		s.view.Reset()
		s.entered = ""
		return s, nil
	case TeaSearchCancel:
		s = s.readOnly()
		s.view.SetValue(s.entered)
		return s, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			return s.enter()
		}
	}

	var cmd tea.Cmd
	s.view, cmd = s.view.Update(msg)
	return s, cmd
}

func (s searchBar) View() string {
	if s.failed != nil && !s.view.Focused() {
		return s.errorStyle.Render(s.view.Prompt + s.failed.Error())
	}
	return s.view.View()
}

// ------------------------- Private

// enter - makes the Search Bar read-only, keeping its regex. Ignored while
// the Search Bar is empty.
func (s searchBar) enter() (tea.Model, tea.Cmd) {
	if s.view.Value() == "" {
		return s, nil
	}

	s = s.readOnly()
	s.entered = s.view.Value()
	s.failed = nil
	return s, func() tea.Msg { return TeaSearchEntered{Regex: s.entered} }
}

// readOnly - stops the Search Bar taking keys, and dims it.
func (s searchBar) readOnly() searchBar {
	s.view.Blur()
	s.view.PromptStyle = readOnlyStyle
	s.view.TextStyle = readOnlyStyle
	return s
}
