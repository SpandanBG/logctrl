package ui

import (
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logTeaCmd string

type uiModel struct {
	ready   bool
	logView viewport.Model
	lines   []string
	stream  reader.Stream
}

func NewUI(stream reader.Stream) (
	app *tea.Program,
	exit func(),
) {
	model := uiModel{
		lines:  make([]string, 0),
		stream: stream,
	}

	app = tea.NewProgram(model, tea.WithAltScreen())

	exit = func() {
		app.Quit()
		defer stream.Close()
	}

	return
}

func (u uiModel) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		u.fetchLog(),
	)
}

func (u uiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return u.executeKeystroke(msg.String())
	case tea.WindowSizeMsg:
		return u.updateViewSize(msg)
	case logTeaCmd:
		return u.refreshLogView(string(msg))
	default:
		return u, nil
	}
}

func (u uiModel) View() string {
	if !u.ready {
		return "loading..."
	}
	return logViewStyle.Render(u.logView.View())
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

func (u uiModel) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	ux := u.updateLogView(size)
	return ux, nil
}

func (u uiModel) refreshLogView(log string) (tea.Model, tea.Cmd) {
	u.lines = append(u.lines, log)
	u.logView.SetContent(
		strings.Join(u.lines, "\n"),
	)
	return u, u.fetchLog()
}

func (u uiModel) fetchLog() tea.Cmd {
	return func() tea.Msg {
		if log := u.stream.Next(); log != "" {
			return logTeaCmd(log)
		}
		return nil
	}
}

func (u uiModel) updateLogView(size tea.WindowSizeMsg) tea.Model {
	size.Width -= logViewStyle.GetHorizontalFrameSize()
	size.Height -= logViewStyle.GetVerticalFrameSize()

	if u.ready {
		u.logView.Width = size.Width
		u.logView.Height = size.Height
	} else {
		u.logView = viewport.New(size.Width, size.Height)
		u.ready = true
	}

	return u
}
