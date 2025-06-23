package ui

import (
	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// focusGroup (fg) specifies the group that is currently focused.
type focusGroup uint

const (
	ROOT_FG focusGroup = iota
	POPUP_FG
)

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logTeaCmd string

type uiModel struct {
	currentFG focusGroup
	logView   tea.Model
}

func NewUI(stream reader.Stream) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		uiModel{
			currentFG: ROOT_FG,
			logView:   components.NewLogView(1, 1, stream),
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
		u.logView.Init(),
	)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return u.executeKeystroke(msg.String())
	}

	switch u.currentFG {
	case ROOT_FG:
		lv, c := u.logView.Update(msg)
		u.logView = lv
		return u, c
	}

	return u, nil
}

func (u uiModel) View() string {
	switch u.currentFG {
	case ROOT_FG:
		return u.logView.View()
	}
	return ""
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
