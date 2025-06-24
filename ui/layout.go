package ui

import (
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// focusGroup (fg) specifies the group that is currently focused.
type focusGroup uint

const (
	LOG_FG focusGroup = iota
	HELP_FG
	// Add all focus group before this comment.
	LOOPBACK

	NEXT_UNIT focusGroup = 1
)

const (
	toolbarSize = 1
)

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logTeaCmd string

type uiModel struct {
	toolbar   tea.Model
	logView   tea.Model
	currentFG focusGroup
}

func NewUI(stream reader.Stream) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		uiModel{
			toolbar:   components.NewToolbar(ui.SizeRatio(1), ui.SizeFixed(toolbarSize)),
			logView:   components.NewLogView(ui.SizeRatio(1), ui.SizeModifier(-toolbarSize), stream),
			currentFG: LOG_FG,
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
	)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return u.executeKeystroke(msg.String())
	}

	cmds := make([]tea.Cmd, 2)
	u.toolbar, cmds[0] = u.toolbar.Update(msg)
	u.logView, cmds[1] = u.logView.Update(msg)

	return u, tea.Batch(cmds...)
}

func (u uiModel) View() string {
	views := make([]string, 2)

	views[0] = u.toolbar.View()
	views[1] = u.logView.View()

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
