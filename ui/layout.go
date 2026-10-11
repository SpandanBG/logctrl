package ui

import (
	"strings"

	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/reader"
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

type uiModel struct {
	toolbar      tea.Model
	logView      tea.Model
	prompt       tea.Model
	help         tea.Model
	promptActive bool
	helpActive   bool
}

func NewUI(stream reader.Stream, keymap []config.Entry) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		newModel(stream, keymap),
		tea.WithAltScreen(),
	)

	exit = func() {
		app.Quit()
		defer stream.Close()
	}

	return
}

func newModel(stream reader.Stream, keymap []config.Entry) uiModel {
	return uiModel{
		toolbar: components.NewToolbar(
			ui.SizeRatio(1),
			ui.SizeFixed(toolbarSize),
			keymap,
		),
		logView: components.NewLogView(
			ui.SizeRatio(1),
			ui.SizeModifier(-toolbarSize),
			stream,
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
	}
}

func (u uiModel) Init() tea.Cmd {
	return tea.Batch(
		u.toolbar.Init(),
		u.logView.Init(),
		u.prompt.Init(),
		u.help.Init(),
	)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if u.promptActive {
			return u.receivePrompt(msg)
		}
		if u.helpActive {
			return u.receiveHelp(msg)
		}
		return u.executeKeystroke(msg)
	}

	return u.batchUpdate(msg)
}

func (u uiModel) View() string {
	return u.batchView()
}

// ------------------------- Private
func (u uiModel) executeKeystroke(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "ctrl+d", "q":
		return u, tea.Quit
	case "tab":
		return u.togglePrompt()
	case "h":
		return u.openHelp()
	default:
		var cmd tea.Cmd
		u.logView, cmd = u.logView.Update(msg)
		return u, cmd
	}
}

func (u uiModel) receivePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return u.togglePrompt()
	default:
		var cmd tea.Cmd
		u.prompt, cmd = u.prompt.Update(msg)
		return u, cmd
	}
}

func (u uiModel) receiveHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		u.helpActive = false
		return u, nil
	case "j", "k":
		var cmd tea.Cmd
		u.help, cmd = u.help.Update(msg)
		return u, cmd
	case "ctrl+c":
		return u, tea.Quit
	default:
		return u, nil
	}
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

	return u, tea.Batch(cmds...)
}

func (u uiModel) batchView() string {
	updateCount := u.getUpdateCount()
	views := make([]string, updateCount)

	views[0] = u.toolbar.View()
	views[1] = u.logView.View()
	if u.helpActive {
		views[1] = u.help.View()
	}

	if u.promptActive {
		views[2] = u.prompt.View()
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

	modifier := toolbarSize
	if u.promptActive {
		modifier += promptSize
	}

	var logViewCmd, promptCmd tea.Cmd

	u.logView, logViewCmd = u.logView.Update(components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-modifier),
	})

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
