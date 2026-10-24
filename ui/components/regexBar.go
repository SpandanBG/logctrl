package components

import (
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Bar - names a regex bar, so the messages it sends say which one it is.
type Bar int

const (
	SearchBar Bar = iota
	FilterBar
)

// barHeight - the rows a shown regex bar takes: its grey top line, then its
// prompt and regex.
const barHeight = 2

// barPrompt - what a regex bar shows before its regex.
type barPrompt struct {
	text       string
	background lipgloss.Color
}

// prompts - the prompt of each regex bar.
var prompts = map[Bar]barPrompt{
	SearchBar: {text: "/:", background: "2"},
	FilterBar: {text: "f:", background: "5"},
}

// ----- Public tea.Msg

// TeaRegexEdit - shows the regex bar, editable, keeping its regex.
type TeaRegexEdit struct{}

// TeaRegexClose - hides the regex bar, emptied, ready for the next regex.
type TeaRegexClose struct{}

// TeaRegexCancel - drops what was typed since the regex was last entered,
// making the regex bar read-only again, or hides it if no regex was entered.
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

// RegexBar - an input for a regex, such as the Search Bar or the Filter Bar:
// hidden, typing, or read-only once a regex is entered, showing that regex or
// why it cannot be used. Typing() implies Shown(); Entered() stays true after
// TeaRegexFailed, until the regex bar is closed.
type RegexBar struct {
	bar           Bar
	width         ui.SizeI
	view          textinput.Model
	shown         bool           // whether the regex bar is shown
	entered       string         // the regex last entered
	failed        error          // why the regex last entered cannot be searched for, if so
	errorWidth    int            // how much of `failed` fits after the prompt
	errorStyle    lipgloss.Style // how `failed` is shown
	editStyle     lipgloss.Style // how the regex is shown while editing
	readOnlyStyle lipgloss.Style // how the regex is shown once entered: dimmed, with no cursor
	promptStyle   lipgloss.Style // how the prompt is shown, in every state
	barStyle      lipgloss.Style // the grey line along the top of the regex bar, as wide as it
}

// NewRegexBar - the regex bar `bar`, hidden, `width` wide, styled by
// `renderer`.
func NewRegexBar(width ui.SizeI, bar Bar, renderer *lipgloss.Renderer) RegexBar {
	view := textinput.New()
	view.Prompt = ""
	return RegexBar{
		bar:           bar,
		width:         width,
		view:          view,
		errorStyle:    renderer.NewStyle().Foreground(lipgloss.Color("1")),
		editStyle:     renderer.NewStyle(),
		readOnlyStyle: renderer.NewStyle().Faint(true),
		promptStyle:   renderer.NewStyle().Foreground(lipgloss.Color("0")).Background(prompts[bar].background),
		barStyle: renderer.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("240")),
	}
}

// Init - asks for the window size, to fit the regex bar to its width.
func (s RegexBar) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update - UpdateBar, as a tea.Model.
func (s RegexBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return s.UpdateBar(msg)
}

// UpdateBar - Update, keeping the RegexBar type, so its state can be read.
func (s RegexBar) UpdateBar(msg tea.Msg) (RegexBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		width := ui.ModifySize(msg, s.width, ui.SizeModifier(0)).Width // only the width is read, so the height is left as it is
		s.barStyle = s.barStyle.Width(width)
		s.errorWidth = width - len(prompts[s.bar].text+" ") // after the prompt and its space
		s.view.Width = s.errorWidth - 1                     // leaving a cell for the cursor
		return s, nil
	case TeaRegexEdit:
		s.shown = true
		s.view.TextStyle = s.editStyle
		return s, s.view.Focus()
	case TeaRegexFailed:
		s.failed = msg.Err
		return s, nil
	case TeaRegexClose:
		return s.close(), nil
	case TeaRegexCancel:
		if s.entered == "" {
			return s.close(), nil
		}
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

// View - the regex bar, or "" while it is hidden.
func (s RegexBar) View() string {
	if !s.shown {
		return ""
	}

	content := s.view.View()
	if s.failed != nil && !s.view.Focused() {
		content = s.errorStyle.Render(ansi.Truncate(s.failed.Error(), s.errorWidth, "…"))
	}
	return s.barStyle.Render(s.promptStyle.Render(prompts[s.bar].text) + " " + content)
}

// Shown - whether the regex bar is shown.
func (s RegexBar) Shown() bool {
	return s.shown
}

// Typing - whether the regex bar is shown and editable.
func (s RegexBar) Typing() bool {
	return s.view.Focused()
}

// Entered - whether a regex has been entered since the regex bar was opened.
func (s RegexBar) Entered() bool {
	return s.entered != ""
}

// Height - the rows the regex bar takes: none while hidden.
func (s RegexBar) Height() int {
	if !s.shown {
		return 0
	}
	return barHeight
}

// ------------------------- Private

// enter - makes the regex bar read-only, keeping its regex. Ignored while
// the regex bar is empty.
func (s RegexBar) enter() (RegexBar, tea.Cmd) {
	if s.view.Value() == "" {
		return s, nil
	}

	s = s.readOnly()
	s.entered = s.view.Value()
	s.failed = nil
	return s, func() tea.Msg { return TeaRegexEntered{Bar: s.bar, Regex: s.entered} }
}

// close - hides the regex bar, emptied.
func (s RegexBar) close() RegexBar {
	s = s.readOnly()
	s.view.Reset()
	s.entered = ""
	s.shown = false
	return s
}

// readOnly - stops the regex bar taking keys, and dims its regex.
func (s RegexBar) readOnly() RegexBar {
	s.view.Blur()
	s.view.TextStyle = s.readOnlyStyle
	return s
}
