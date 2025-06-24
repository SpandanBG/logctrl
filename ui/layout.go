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
	currentFG focusGroup
	logFG     []tea.Model
	helpFG    []tea.Model
}

func NewUI(stream reader.Stream) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		uiModel{
			toolbar:   components.NewToolbar(ui.SizeRatio(1), ui.SizeFixed(toolbarSize)),
			currentFG: LOG_FG,
			logFG: []tea.Model{
				components.NewLogView(ui.SizeRatio(1), ui.SizeModifier(-toolbarSize), stream),
			},
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
	var fg []tea.Model

	switch u.currentFG {
	case LOG_FG:
		fg = u.logFG
	case HELP_FG:
		fg = u.helpFG
	default:
		return nil
	}

	var cmds []tea.Cmd
	for _, each := range fg {
		cmds = append(cmds, each.Init())
	}

	return tea.Batch(cmds...)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return u.executeKeystroke(msg.String())
	}

	cmds := []tea.Cmd{nil}
	u.toolbar, cmds[0] = u.toolbar.Update(msg)

	switch u.currentFG {
	case LOG_FG:
		u.logFG, cmds = u.batchUpdate(msg, u.logFG)
	case HELP_FG:
		u.helpFG, cmds = u.batchUpdate(msg, u.helpFG)
	default:
		return u, nil
	}

	return u, tea.Batch(cmds...)
}

func (u uiModel) View() string {
	var fg []tea.Model
	switch u.currentFG {
	case LOG_FG:
		fg = u.logFG
	case HELP_FG:
		fg = u.helpFG
	default:
		return ""
	}

	views := make([]string, len(fg)+1)
	views[0] = u.toolbar.View()
	for i, each := range fg {
		views[i+1] = each.View()
	}

	return strings.Join(views, "\n")
}

// ------------------------- Private
func (u uiModel) executeKeystroke(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "ctrl+d", "q":
		return u, tea.Quit
	case "tab":
		return u.nextFocusGroup()
	default:
		return u, nil
	}
}

func (u uiModel) batchUpdate(msg tea.Msg, fg []tea.Model) ([]tea.Model, []tea.Cmd) {
	cmds := make([]tea.Cmd, len(fg))
	for i, each := range fg {
		fg[i], cmds[i] = each.Update(msg)
	}
	return fg, cmds
}

func (u uiModel) nextFocusGroup() (tea.Model, tea.Cmd) {
	// Go to next focus group
	u.currentFG = (u.currentFG + NEXT_UNIT) % LOOPBACK

	var cmds []tea.Cmd
	switch u.currentFG {
	case LOG_FG:
		u.logFG, cmds = u.batchUpdate(nil, u.logFG)
	case HELP_FG:
		u.helpFG, cmds = u.batchUpdate(nil, u.helpFG)
	default:
		return u, nil
	}

	return u, tea.Batch(cmds...)
}
