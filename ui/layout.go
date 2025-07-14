package ui

import (
	"strings"

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
	promptActive bool
}

func NewUI(stream reader.StreamV2) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		uiModel{
			toolbar: components.NewToolbar(
				ui.SizeRatio(1),
				ui.SizeFixed(toolbarSize),
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
		},
		tea.WithAltScreen(),
	)

	exit = func() {
		app.Quit()
		defer stream.Close()
	}

	return
}

func (u uiModel) Init() tea.Cmd {
	return tea.Batch(
		u.toolbar.Init(),
		u.logView.Init(),
		u.prompt.Init(),
	)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if u.promptActive {
			return u.receivePrompt(msg)
		}
		return u.executeKeystroke(msg.String())
	}

	return u.batchUpdate(msg)
}

func (u uiModel) View() string {
	return u.batchView()
}

// ------------------------- Private
func (u uiModel) executeKeystroke(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "ctrl+d", "q":
		return u, tea.Quit
	case "tab":
		return u.togglePrompt()
	default:
		return u, nil
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

func (u uiModel) batchUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateCount := u.getUpdateCount()
	cmds := make([]tea.Cmd, updateCount)

	u.toolbar, cmds[0] = u.toolbar.Update(msg)
	u.logView, cmds[1] = u.logView.Update(msg)

	if u.promptActive {
		u.prompt, cmds[2] = u.prompt.Update(msg)
	}

	return u, tea.Batch(cmds...)
}

func (u uiModel) batchView() string {
	updateCount := u.getUpdateCount()
	views := make([]string, updateCount)

	views[0] = u.toolbar.View()
	views[1] = u.logView.View()

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
