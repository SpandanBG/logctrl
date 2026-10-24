package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
var prompts = map[Bar]string{SearchBar: "/:", FilterBar: "f:"}

// promptColors - the background of each regex bar's prompt.
var promptColors = map[Bar]lipgloss.Color{SearchBar: "2", FilterBar: "5"}

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
	bar         Bar
	width       ui.SizeI
	view        textinput.Model
	entered     string         // the regex last entered
	failed      error          // why the regex last entered cannot be searched for, if so
	errorWidth  int            // how much of `failed` fits after the prompt
	errorStyle  lipgloss.Style // how `failed` is shown
	promptStyle lipgloss.Style // how the prompt is shown, in every state
	barStyle    lipgloss.Style // the grey line along the top of the regex bar, as wide as it
}

// NewRegexBar - the one-line regex bar `bar`, `width` wide, styled by
// `renderer`.
func NewRegexBar(width ui.SizeI, bar Bar, renderer *lipgloss.Renderer) tea.Model {
	view := textinput.New()
	view.Prompt = ""
	return regexBar{
		bar:         bar,
		width:       width,
		view:        view,
		errorStyle:  renderer.NewStyle().Foreground(lipgloss.Color("1")),
		promptStyle: renderer.NewStyle().Foreground(lipgloss.Color("0")).Background(promptColors[bar]),
		barStyle: renderer.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("240")),
	}
}

func (s regexBar) Init() tea.Cmd {
	return tea.WindowSize()
}

func (s regexBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		width := ui.ModifySize(msg, s.width, ui.SizeFixed(1)).Width
		s.barStyle = s.barStyle.Width(width)
		s.errorWidth = width - len(prompts[s.bar]+" ") // after the prompt and its space
		s.view.Width = s.errorWidth - 1                // leaving a cell for the cursor
		return s, nil
	case TeaRegexEdit:
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
	prompt := s.promptStyle.Render(prompts[s.bar]) + " "
	if s.failed != nil && !s.view.Focused() {
		return s.barStyle.Render(prompt + s.errorStyle.Render(ansi.Truncate(s.failed.Error(), s.errorWidth, "…")))
	}
	return s.barStyle.Render(prompt + s.view.View())
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

// readOnly - stops the regex bar taking keys, and dims its regex.
func (s regexBar) readOnly() regexBar {
	s.view.Blur()
	s.view.TextStyle = readOnlyStyle
	return s
}
