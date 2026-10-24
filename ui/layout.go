package ui

import (
	"slices"
	"strings"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	toolbarSize = 1
	promptSize  = 16
)

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logTeaCmd string

// closeActions - the Action that closes each regex bar, by its
// components.Bar.
var closeActions = [...]config.Action{
	components.SearchBar: config.CloseSearch,
	components.FilterBar: config.CloseFilter,
}

// barOrder - the regex bars from the top, under the Log View: the Filter Bar
// above the Search Bar.
var barOrder = []components.Bar{components.FilterBar, components.SearchBar}

// uiModel - the app: the Toolbar over the Log View (or the Help), then the
// regex bars and the Prompt while they are shown.
type uiModel struct {
	stream       cmd.Stream
	toolbar      tea.Model
	logView      tea.Model
	prompt       tea.Model
	help         tea.Model
	bars         [len(closeActions)]components.RegexBar // each regex bar, by its components.Bar
	actionOf     map[string]config.Action               // the Action each key in the Keymap triggers
	promptActive bool
	helpActive   bool
	regex        string // the regex being searched for; "" if none
}

// NewUI - the app over `stream`, keyed by `keymap`, styled by `renderer`.
func NewUI(stream cmd.Stream, keymap []config.Entry, renderer *lipgloss.Renderer) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		newModel(stream, keymap, renderer),
		tea.WithAltScreen(),
	)

	exit = func() {
		app.Quit()
		defer stream.Close()
	}

	return
}

func newModel(stream cmd.Stream, keymap []config.Entry, renderer *lipgloss.Renderer) uiModel {
	actionOf := map[string]config.Action{}
	for _, entry := range keymap {
		actionOf[entry.Key] = entry.Action
	}

	return uiModel{
		stream:   stream,
		actionOf: actionOf,
		toolbar: components.NewToolbar(
			ui.SizeRatio(1),
			ui.SizeFixed(toolbarSize),
			keymap,
		),
		logView: components.NewLogView(
			ui.SizeRatio(1),
			ui.SizeModifier(-toolbarSize),
			stream,
			renderer,
		),
		prompt: components.NewPrompt(
			ui.SizeRatio(1),
			ui.SizeFixed(promptSize),
		),
		help: components.NewHelp(
			ui.SizeRatio(1),
			ui.SizeModifier(-toolbarSize),
			keymap,
		),
		bars: [len(closeActions)]components.RegexBar{
			components.SearchBar: components.NewRegexBar(ui.SizeRatio(1), components.SearchBar, renderer),
			components.FilterBar: components.NewRegexBar(ui.SizeRatio(1), components.FilterBar, renderer),
		},
	}
}

func (u uiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		u.toolbar.Init(),
		u.logView.Init(),
		u.prompt.Init(),
		u.help.Init(),
	}
	for _, bar := range u.bars {
		cmds = append(cmds, bar.Init())
	}
	return tea.Batch(cmds...)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if u.promptActive {
			return u.receivePrompt(msg)
		}
		if slices.Contains(config.Reserved, msg.String()) {
			return u, tea.Quit
		}
		for which, bar := range u.bars {
			if bar.Typing() {
				return u.receiveBar(components.Bar(which), msg)
			}
		}
		if u.helpActive {
			return u.receiveHelp(msg)
		}
		return u.executeKeystroke(msg)
	case components.TeaRegexEntered:
		if msg.Bar == components.FilterBar || msg.Regex == u.regex {
			return u, nil
		}
		u.regex = msg.Regex
		if err := u.stream.Update(cmd.SearchFor{Regex: msg.Regex}); err != nil {
			u.regex = ""
			u, _ = u.updateBar(components.SearchBar, components.TeaRegexFailed{Err: err})
		}
		return u.followLogView()
	}

	return u.batchUpdate(msg)
}

func (u uiModel) View() string {
	return u.batchView()
}

// ------------------------- Private
func (u uiModel) executeKeystroke(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch u.actionOf[msg.String()] {
	case config.Quit:
		return u, tea.Quit
	case config.OpenHelp:
		return u.openHelp()
	case config.OpenPrompt:
		return u.togglePrompt()
	case config.ScrollUp:
		return u.scrollLogView(-1)
	case config.ScrollDown:
		return u.scrollLogView(1)
	case config.Follow:
		u.stream.Update(cmd.Follow{}) // before Following, so no scan Pins the Window after it
		return u.followLogView()
	case config.OpenSearch:
		return u.openBar(components.SearchBar)
	case config.OpenFilter:
		return u.openBar(components.FilterBar)
	case config.CloseFilter:
		return u.closeBar(components.FilterBar)
	case config.CloseSearch:
		return u.closeBar(components.SearchBar)
	case config.PreviousMatch:
		return u.step(cmd.PreviousMatch{})
	case config.NextMatch:
		return u.step(cmd.NextMatch{})
	}
	return u, nil
}

// step - while a regex is entered, steps to the Line with a Match `msg` asks
// for.
func (u uiModel) step(msg any) (tea.Model, tea.Cmd) {
	if !u.bars[components.SearchBar].Entered() {
		return u, nil
	}
	return u.updateStream(msg)
}

// updateStream - hands `msg` to the Stream in the background, as it may scan
// the History, then redraws the Log View.
func (u uiModel) updateStream(msg any) (tea.Model, tea.Cmd) {
	stream := u.stream
	return u, func() tea.Msg {
		stream.Update(msg)
		return components.TeaLogRefresh{}
	}
}

func (u uiModel) followLogView() (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	u.logView, cmd = u.logView.Update(components.TeaFollow{})
	return u, cmd
}

func (u uiModel) scrollLogView(n int) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	u.logView, cmd = u.logView.Update(components.TeaScroll(n))
	return u, cmd
}

func (u uiModel) receivePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if u.actionOf[msg.String()] == config.Close {
		return u.togglePrompt()
	}

	var cmd tea.Cmd
	u.prompt, cmd = u.prompt.Update(msg)
	return u, cmd
}

func (u uiModel) receiveHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch u.actionOf[msg.String()] {
	case config.Close:
		u.helpActive = false
		return u, nil
	case config.ScrollUp:
		return u.scrollHelp(-1)
	case config.ScrollDown:
		return u.scrollHelp(1)
	}
	return u, nil
}

func (u uiModel) scrollHelp(n int) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	u.help, cmd = u.help.Update(components.TeaScroll(n))
	return u, cmd
}

func (u uiModel) batchUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateCount := u.getUpdateCount()
	cmds := make([]tea.Cmd, updateCount)

	u.toolbar, cmds[0] = u.toolbar.Update(msg)
	u.logView, cmds[1] = u.logView.Update(msg)

	if u.promptActive {
		u.prompt, cmds[2] = u.prompt.Update(msg)
	}

	var helpCmd tea.Cmd
	u.help, helpCmd = u.help.Update(msg)
	cmds = append(cmds, helpCmd)
	for which := range u.bars {
		var barCmd tea.Cmd
		u, barCmd = u.updateBar(components.Bar(which), msg)
		cmds = append(cmds, barCmd)
	}

	return u, tea.Batch(cmds...)
}

func (u uiModel) batchView() string {
	views := []string{u.toolbar.View(), u.logView.View()}
	if u.helpActive {
		views[1] = u.help.View()
	}

	for _, which := range barOrder {
		if u.bars[which].Shown() {
			views = append(views, u.bars[which].View())
		}
	}

	if u.promptActive {
		views = append(views, u.prompt.View())
	}

	return strings.Join(views, "\n")
}

func (u uiModel) getUpdateCount() int {
	updateCount := 2
	if u.promptActive {
		updateCount += 1
	}

	return updateCount
}

func (u uiModel) togglePrompt() (tea.Model, tea.Cmd) {
	u.promptActive = !u.promptActive

	var logViewCmd, promptCmd tea.Cmd
	u, logViewCmd = u.resizePanels()

	u.prompt, promptCmd = u.prompt.Update(components.TeaPromptToggle{
		BringFocus: u.promptActive,
	})

	return u, tea.Batch(
		logViewCmd,
		promptCmd,
	)
}

// openHelp - shows the Help, from its first row, in place of the Log View.
func (u uiModel) openHelp() (tea.Model, tea.Cmd) {
	u.helpActive = true

	var cmd tea.Cmd
	u.help, cmd = u.help.Update(components.TeaHelpOpen{})
	return u, cmd
}

// openBar - shows `which`, editable, under the Log View.
func (u uiModel) openBar(which components.Bar) (tea.Model, tea.Cmd) {
	var logViewCmd, barCmd tea.Cmd
	u, barCmd = u.updateBar(which, components.TeaRegexEdit{})
	u, logViewCmd = u.resizePanels()
	return u, tea.Batch(logViewCmd, barCmd)
}

// receiveBar - hands `msg` to `which`, which is being typed in, unless it
// closes `which` or cancels typing in it.
func (u uiModel) receiveBar(which components.Bar, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch u.actionOf[msg.String()] {
	case closeActions[which]:
		return u.closeBar(which)
	case config.Close:
		return u.cancelBar(which)
	}
	return u.updateBar(which, msg)
}

// closeBar - removes `which`, emptied, giving its rows back to the Log View;
// closing the Search Bar ends the Search.
func (u uiModel) closeBar(which components.Bar) (tea.Model, tea.Cmd) {
	if which == components.SearchBar {
		u = u.endSearch()
	}
	u, _ = u.updateBar(which, components.TeaRegexClose{})
	return u.resizePanels()
}

// cancelBar - drops what was typed in `which`: back to the regex last
// entered, or no `which` if none was.
func (u uiModel) cancelBar(which components.Bar) (tea.Model, tea.Cmd) {
	u, _ = u.updateBar(which, components.TeaRegexCancel{})
	if u.bars[which].Shown() {
		return u, nil
	}
	return u.closeBar(which) // now hidden: closing it gives its rows back, and ends a Search
}

// endSearch - ends the Search: once a regex has been entered, the Window
// Follows; until then, the Log View redraws as it was.
func (u uiModel) endSearch() uiModel {
	u.regex = ""
	u.stream.Update(cmd.SearchFor{}) // ending a Search never fails; before Following, so no scan Pins the Window after it
	if u.bars[components.SearchBar].Entered() {
		u.logView, _ = u.logView.Update(components.TeaFollow{})
	} else {
		u.logView, _ = u.logView.Update(components.TeaLogRefresh{})
	}
	return u
}

// updateBar - hands `msg` to the regex bar `which`.
func (u uiModel) updateBar(which components.Bar, msg tea.Msg) (uiModel, tea.Cmd) {
	var cmd tea.Cmd
	u.bars[which], cmd = u.bars[which].UpdateBar(msg)
	return u, cmd
}

// resizePanels - fits the Log View into the rows the Toolbar, the Filter Bar,
// the Search Bar and the Prompt leave, and the Help into the rows the Toolbar,
// the Filter Bar and the Search Bar leave.
func (u uiModel) resizePanels() (uiModel, tea.Cmd) {
	modifier := toolbarSize
	for _, bar := range u.bars {
		modifier += bar.Height()
	}

	var logViewCmd, helpCmd tea.Cmd
	u.help, helpCmd = u.help.Update(components.TeaHelpSizeUpdate{
		Height: ui.SizeModifier(-modifier),
	})

	if u.promptActive {
		modifier += promptSize
	}
	u.logView, logViewCmd = u.logView.Update(components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-modifier),
	})
	return u, tea.Batch(logViewCmd, helpCmd)
}
