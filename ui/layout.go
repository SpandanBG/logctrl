package ui

import (
	"slices"
	"strings"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/headless"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	toolbarSize = 1  // the rows the Toolbar takes
	promptSize  = 16 // the rows the Prompt takes while open
)

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
	logs         *headless.Logs                         // the Window the Log View shows, and the Passes run over it
	toolbar      tea.Model                              // the Toolbar, along the top
	logView      tea.Model                              // the Log View, under the Toolbar
	prompt       tea.Model                              // the Prompt, at the bottom while open
	help         tea.Model                              // the Help, in place of the Log View while open
	bars         [len(closeActions)]components.RegexBar // each regex bar, by its components.Bar
	actionOf     map[string]config.Action               // the Action each key in the Keymap triggers
	promptActive bool                                   // the Prompt is open, and keys go to it
	helpActive   bool                                   // the Help is open, in place of the Log View
}

// NewUI - the app over `logs`, keyed by `keymap`, styled by `renderer`.
func NewUI(logs *headless.Logs, keymap []config.Entry, renderer *lipgloss.Renderer) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		newModel(logs, keymap, renderer),
		tea.WithAltScreen(),
	)

	exit = func() {
		app.Quit()
		defer logs.Close()
	}

	return
}

// Init - starts every panel: each asks for the terminal's size, and the Log
// View waits for the Window to change.
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

// Update - hands a key to the Prompt while it is open; else a Reserved key
// Quits, and a key goes to a regex bar being typed in, the Help, or the Action
// it triggers. A regex entered starts or ends a Search or a Filter; any other
// message goes to every panel.
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
		if msg.Bar == components.FilterBar {
			if err := u.logs.Update(cmd.FilterFor{Regex: msg.Regex}); err != nil {
				u, _ = u.updateBar(components.FilterBar, components.TeaRegexFailed{Err: err})
			}
			return u, nil
		}
		if err := u.logs.Update(cmd.SearchFor{Regex: msg.Regex}); err != nil {
			u, _ = u.updateBar(components.SearchBar, components.TeaRegexFailed{Err: err})
		}
		return u, nil
	}

	return u.batchUpdate(msg)
}

// View - the panels shown, top to bottom.
func (u uiModel) View() string {
	return u.batchView()
}

// ------------------------- Private

// newModel - the app's model over `logs`, keyed by `keymap`, styled by
// `renderer`, with the Help, the Prompt and the regex bars closed.
func newModel(logs *headless.Logs, keymap []config.Entry, renderer *lipgloss.Renderer) uiModel {
	actionOf := map[string]config.Action{}
	for _, entry := range keymap {
		actionOf[entry.Key] = entry.Action
	}

	return uiModel{
		logs:     logs,
		actionOf: actionOf,
		toolbar: components.NewToolbar(
			ui.SizeRatio(1),
			ui.SizeFixed(toolbarSize),
			keymap,
		),
		logView: components.NewLogView(
			ui.SizeRatio(1),
			ui.SizeModifier(-toolbarSize),
			logs,
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

// executeKeystroke - does the Action the key `msg` triggers in the Keymap.
func (u uiModel) executeKeystroke(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch u.actionOf[msg.String()] {
	case config.Quit:
		return u, tea.Quit
	case config.OpenHelp:
		return u.openHelp()
	case config.OpenPrompt:
		return u.togglePrompt()
	case config.ScrollUp:
		u.scroll(-1)
	case config.ScrollDown:
		u.scroll(1)
	case config.Follow:
		u.follow()
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
	return u.updateLogs(msg)
}

// updateLogs - hands `msg` to the Logs in the background, as it may scan
// the History; the Log View redraws on the notice that follows.
func (u uiModel) updateLogs(msg any) (tea.Model, tea.Cmd) {
	logs := u.logs
	return u, func() tea.Msg {
		logs.Update(msg) // stepping never fails
		return nil
	}
}

// follow - makes the Window Following, dropping the Current Line.
func (u uiModel) follow() {
	u.logs.Update(cmd.Follow{}) // Following never fails
}

// scroll - Scrolls the Window `n` Lines.
func (u uiModel) scroll(n int) {
	u.logs.Update(cmd.Scroll{N: n}) // Scrolling never fails
}

// receivePrompt - closes the Prompt on Close, else types the key `msg` in it.
func (u uiModel) receivePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if u.actionOf[msg.String()] == config.Close {
		return u.togglePrompt()
	}

	var cmd tea.Cmd
	u.prompt, cmd = u.prompt.Update(msg)
	return u, cmd
}

// receiveHelp - closes the Help on Close and scrolls it on Scroll up or down;
// any other key does nothing while it is open.
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

// scrollHelp - scrolls the Help `n` rows: negative is up, positive is down.
func (u uiModel) scrollHelp(n int) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	u.help, cmd = u.help.Update(components.TeaHelpScroll(n))
	return u, cmd
}

// batchUpdate - hands `msg` to every panel, the Prompt only while it is open.
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

// batchView - the Toolbar, the Log View or the Help, then the regex bars and
// the Prompt that are shown, joined top to bottom.
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

// getUpdateCount - how many panels batchUpdate updates first: the Toolbar and
// the Log View, and the Prompt while it is open.
func (u uiModel) getUpdateCount() int {
	updateCount := 2
	if u.promptActive {
		updateCount += 1
	}

	return updateCount
}

// togglePrompt - opens the Prompt, focused, or closes it, resizing the Log
// View to fit.
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
// closing the Search Bar ends the Search, and closing the Filter Bar the
// Filter.
func (u uiModel) closeBar(which components.Bar) (tea.Model, tea.Cmd) {
	switch which {
	case components.SearchBar:
		u = u.endSearch()
	case components.FilterBar:
		u.logs.Update(cmd.FilterFor{}) // ending a Filter never fails
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
// Follows; until then, it keeps its mode.
func (u uiModel) endSearch() uiModel {
	u.logs.Update(cmd.SearchFor{}) // ending a Search never fails
	if u.bars[components.SearchBar].Entered() {
		u.follow()
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
