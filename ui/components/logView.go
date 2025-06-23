package components

import (
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type teaLogCmd string

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logView struct {
	wRatio float32        // how much % of the total width to span (value b/w: 0-1)
	hRatio float32        // how much % of the total height to span (value b/w: 0-1)
	view   viewport.Model // holds the viewport
	lines  []string       // holds a window of the logs lines
	stream reader.Stream  // log feed stream to be displayed
	ready  bool           // if `true` the viewport is ready to render
}

func NewLogView(wRatio, hRatio float32, stream reader.Stream) tea.Model {
	return logView{
		wRatio: wRatio,
		hRatio: hRatio,
		stream: stream,
	}
}

func (l logView) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		l.fetchLog(),
	)
}

func (l logView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return l.executeKey(msg.String())
	case tea.WindowSizeMsg:
		return l.updateViewSize(msg)
	case teaLogCmd:
		return l.refreshView(string(msg))
	}
	return l, nil
}

func (l logView) View() string {
	return logViewStyle.Render(l.view.View())
}

// ------------------------- Private
func (l logView) executeKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	default:
		return l, nil
	}
}

func (l logView) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	w := int(l.wRatio*float32(size.Width)) - logViewStyle.GetHorizontalFrameSize()
	h := int(l.hRatio*float32(size.Height)) - logViewStyle.GetVerticalFrameSize()

	if l.ready {
		l.view.Width = w
		l.view.Height = h
	} else {
		l.view = viewport.New(w, h)
		l.ready = true
	}

	return l, nil
}

func (l logView) refreshView(log string) (tea.Model, tea.Cmd) {
	l.lines = append(l.lines, log)
	l.view.SetContent(
		strings.Join(l.lines, "\n"),
	)
	return l, l.fetchLog()
}

func (l logView) fetchLog() tea.Cmd {
	return func() tea.Msg {
		if log := l.stream.Next(); log != "" {
			return teaLogCmd(log)
		}
		return nil
	}
}
