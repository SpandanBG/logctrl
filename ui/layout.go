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
	toolbarSize   = 1
	searchBarSize = 1
	promptSize    = 16
)

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logTeaCmd string

type uiModel struct {
	stream        cmd.Stream
	toolbar       tea.Model
	logView       tea.Model
	prompt        tea.Model
	help          tea.Model
	searchBar     tea.Model
	actionOf      map[string]config.Action // the Action each key in the Keymap triggers
	promptActive  bool
	helpActive    bool
	searchActive  bool   // the Search Bar is shown
	searchTyping  bool   // the Search Bar is shown and editable
	searchEntered bool   // a regex has been entered in the Search Bar
	regex         string // the regex being searched for; "" if none
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
		searchBar: components.NewSearchBar(ui.SizeRatio(1), renderer),
	}
}

func (u uiModel) Init() tea.Cmd {
	return tea.Batch(
		u.toolbar.Init(),
		u.logView.Init(),
		u.prompt.Init(),
		u.help.Init(),
		u.searchBar.Init(),
	)
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
		if u.searchTyping {
			return u.receiveSearch(msg)
		}
		if u.helpActive {
			return u.receiveHelp(msg)
		}
		return u.executeKeystroke(msg)
	case components.TeaSearchEntered:
		u.searchTyping = false
		u.searchEntered = true
		if msg.Regex == u.regex {
			return u, nil
		}
		u.regex = msg.Regex
		if err := u.stream.Update(cmd.SearchFor{Regex: msg.Regex}); err != nil {
			u.regex = ""
			u.searchBar, _ = u.searchBar.Update(components.TeaSearchFailed{Err: err})
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
		return u.openSearch()
	case config.CloseSearch:
		return u.closeSearch()
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
	if !u.searchEntered {
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

	var helpCmd, searchBarCmd tea.Cmd
	u.help, helpCmd = u.help.Update(msg)
	u.searchBar, searchBarCmd = u.searchBar.Update(msg)
	cmds = append(cmds, helpCmd, searchBarCmd)

	return u, tea.Batch(cmds...)
}

func (u uiModel) batchView() string {
	views := []string{u.toolbar.View(), u.logView.View()}
	if u.helpActive {
		views[1] = u.help.View()
	}

	if u.searchActive {
		views = append(views, u.searchBar.View())
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

// openSearch - shows the Search Bar, editable, under the Log View.
func (u uiModel) openSearch() (tea.Model, tea.Cmd) {
	u.searchActive = true
	u.searchTyping = true

	var logViewCmd, searchBarCmd tea.Cmd
	u, logViewCmd = u.resizePanels()
	u.searchBar, searchBarCmd = u.searchBar.Update(components.TeaSearchEdit{})
	return u, tea.Batch(logViewCmd, searchBarCmd)
}

func (u uiModel) receiveSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch u.actionOf[msg.String()] {
	case config.CloseSearch:
		return u.closeSearch()
	case config.Close:
		return u.cancelSearch()
	}

	var cmd tea.Cmd
	u.searchBar, cmd = u.searchBar.Update(msg)
	return u, cmd
}

// resizePanels - fits the Log View into the rows the Toolbar, the Search Bar
// and the Prompt leave, and the Help into the rows the Toolbar and the Search
// Bar leave.
func (u uiModel) resizePanels() (uiModel, tea.Cmd) {
	modifier := toolbarSize
	if u.searchActive {
		modifier += searchBarSize
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

// closeSearch - removes the Search Bar, giving its row back to the Log View.
func (u uiModel) closeSearch() (tea.Model, tea.Cmd) {
	u.searchActive = false
	u.searchTyping = false
	u.searchEntered = false
	u.regex = ""
	u.stream.Update(cmd.SearchFor{})
	u.logView, _ = u.logView.Update(components.TeaLogRefresh{})

	var cmd tea.Cmd
	u, cmd = u.resizePanels()
	u.searchBar, _ = u.searchBar.Update(components.TeaSearchClose{})
	return u, cmd
}

// cancelSearch - drops what was typed in the Search Bar: back to the regex
// last entered, or no Search Bar if none was.
func (u uiModel) cancelSearch() (tea.Model, tea.Cmd) {
	if !u.searchEntered {
		return u.closeSearch()
	}

	u.searchTyping = false
	u.searchBar, _ = u.searchBar.Update(components.TeaSearchCancel{})
	return u, nil
}
