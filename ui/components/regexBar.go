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

// Bar - names a regex bar, so the messages it sends say which one it is.
type Bar int

const (
	SearchBar Bar = iota
	FilterBar
)

// prompts - what each regex bar shows before its regex.
var prompts = map[Bar]string{SearchBar: "/", FilterBar: "f"}

// ----- Public tea.Msg

// TeaRegexEdit - makes the regex bar editable, keeping its regex.
type TeaRegexEdit struct{}

// TeaRegexClose - empties the regex bar, ready for the next regex.
type TeaRegexClose struct{}

// TeaRegexCancel - drops what was typed since the regex was last entered,
// making the regex bar read-only again.
type TeaRegexCancel struct{}

// TeaRegexEntered - the user entered `Regex` in `Bar`, now read-only.
type TeaRegexEntered struct {
	Bar   Bar
	Regex string
}

// TeaRegexFailed - the regex last entered cannot be used, because of `Err`.
type TeaRegexFailed struct {
	Err error
}

// regexBar - a one-line input for a regex, such as the Search Bar or the
// Filter Bar.
type regexBar struct {
	bar        Bar
	width      ui.SizeI
	view       textinput.Model
	entered    string         // the regex last entered
	failed     error          // why the regex last entered cannot be searched for, if so
	errorStyle lipgloss.Style // how `failed` is shown
}

// NewRegexBar - the one-line regex bar `bar`, `width` wide, styled by
// `renderer`.
func NewRegexBar(width ui.SizeI, bar Bar, renderer *lipgloss.Renderer) tea.Model {
	view := textinput.New()
	view.Prompt = prompts[bar]
	return regexBar{
		bar:        bar,
		width:      width,
		view:       view,
		errorStyle: renderer.NewStyle().Foreground(lipgloss.Color("1")),
	}
}

func (s regexBar) Init() tea.Cmd {
	return tea.WindowSize()
}

func (s regexBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.view.Width = ui.ModifySize(msg, s.width, ui.SizeFixed(1)).Width - len(s.view.Prompt) - 1
		return s, nil
	case TeaRegexEdit:
		s.view.PromptStyle = lipgloss.NewStyle()
		s.view.TextStyle = lipgloss.NewStyle()
		return s, s.view.Focus()
	case TeaRegexFailed:
		s.failed = msg.Err
		return s, nil
	case TeaRegexClose:
		s = s.readOnly()
		s.view.Reset()
		s.entered = ""
		return s, nil
	case TeaRegexCancel:
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

func (s regexBar) View() string {
	if s.failed != nil && !s.view.Focused() {
		return s.errorStyle.Render(s.view.Prompt + s.failed.Error())
	}
	return s.view.View()
}

// ------------------------- Private

// enter - makes the regex bar read-only, keeping its regex. Ignored while
// the regex bar is empty.
func (s regexBar) enter() (tea.Model, tea.Cmd) {
	if s.view.Value() == "" {
		return s, nil
	}

	s = s.readOnly()
	s.entered = s.view.Value()
	s.failed = nil
	return s, func() tea.Msg { return TeaRegexEntered{Bar: s.bar, Regex: s.entered} }
}

// readOnly - stops the regex bar taking keys, and dims it.
func (s regexBar) readOnly() regexBar {
	s.view.Blur()
	s.view.PromptStyle = readOnlyStyle
	s.view.TextStyle = readOnlyStyle
	return s
}
