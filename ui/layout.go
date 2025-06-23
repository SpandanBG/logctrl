package ui

import (
	"strings"

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
	rootFG    []tea.Model
	popupFG   []tea.Model
}

func NewUI(stream reader.Stream) (
	app *tea.Program,
	exit func(),
) {
	app = tea.NewProgram(
		uiModel{
			currentFG: ROOT_FG,
			rootFG: []tea.Model{
				components.NewLogView(1, 1, stream),
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
	case ROOT_FG:
		fg = u.rootFG
	case POPUP_FG:
		fg = u.popupFG
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

	var cmds []tea.Cmd
	switch u.currentFG {
	case ROOT_FG:
		u.rootFG, cmds = u.batchUpdate(msg, u.rootFG)
	case POPUP_FG:
		u.popupFG, cmds = u.batchUpdate(msg, u.popupFG)
	default:
		return u, nil
	}

	return u, tea.Batch(cmds...)
}

func (u uiModel) View() string {
	var fg []tea.Model
	switch u.currentFG {
	case ROOT_FG:
		fg = u.rootFG
	case POPUP_FG:
		fg = u.popupFG
	default:
		return ""
	}

	var views []string
	for _, each := range fg {
		views = append(views, each.View())
	}

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

func (u uiModel) batchUpdate(msg tea.Msg, fg []tea.Model) ([]tea.Model, []tea.Cmd) {
	var cmds []tea.Cmd
	for i, each := range fg {
		vx, cmd := each.Update(msg)
		fg[i] = vx
		cmds = append(cmds, cmd)
	}

	return fg, cmds
}
