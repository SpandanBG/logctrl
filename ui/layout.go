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
	toolbar tea.Model
	logView tea.Model
	prompt  tea.Model
}

func NewUI(stream reader.Stream) (
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
				ui.SizeModifier(-(toolbarSize + promptSize)),
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
		return u.executeKeystroke(msg.String())
	}

	cmds := make([]tea.Cmd, 3)
	u.toolbar, cmds[0] = u.toolbar.Update(msg)
	u.logView, cmds[1] = u.logView.Update(msg)
	u.prompt, cmds[2] = u.prompt.Update(msg)

	return u, tea.Batch(cmds...)
}

func (u uiModel) View() string {
	views := make([]string, 3)

	views[0] = u.toolbar.View()
	views[1] = u.logView.View()
	views[2] = u.prompt.View()

	return strings.Join(views, "\n")
}

// ------------------------- Private
func (u uiModel) executeKeystroke(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "ctrl+d", "q":
		return u, tea.Quit
	default:
		return u, nil
	}
}
