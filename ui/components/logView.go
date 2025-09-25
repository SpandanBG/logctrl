package components

import (
	"github.com/SpandanBG/logctrl/reader"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ----- Public tea.Msg
type TeaLogSizeUpdate struct {
	Width  ui.SizeI
	Height ui.SizeI
}

// ----- Private tea.Msg
type teaLogCmd string

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logView struct {
	width   ui.SizeI        // width modifier
	height  ui.SizeI        // height modifier
	view    viewport.Model  // holds the viewport
	stream  reader.StreamV2 // log feed stream to be displayed
	ready   bool            // if `true` the viewport is ready to render
	nextLog chan bool       // notification channel from stream
}

func NewLogView(width, height ui.SizeI, stream reader.StreamV2) tea.Model {
	nextLog := make(chan bool)

	stream.SetBufferSize(1)
	stream.Start(nextLog)

	return logView{
		width:   width,
		height:  height,
		stream:  stream,
		nextLog: nextLog,
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
	case TeaLogSizeUpdate:
		return l.updateSize(msg)
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
	// get relative size of view
	size = ui.ModifySize(size, l.width, l.height)
	w := size.Width - logViewStyle.GetHorizontalFrameSize()
	h := size.Height - logViewStyle.GetVerticalFrameSize()

	// update view width and height
	if l.ready {
		l.view.Width = w
		l.view.Height = h
	} else {
		l.view = viewport.New(w, h)
		l.ready = true
	}

	// set buffer size to the hight of the screen
	l.stream.SetBufferSize(h)

	return l, nil
}

func (l logView) refreshView(logs string) (tea.Model, tea.Cmd) {
	l.view.SetContent(logs)
	return l, l.fetchLog()
}

func (l logView) fetchLog() tea.Cmd {
	return func() tea.Msg {
		<-l.nextLog
		return teaLogCmd(l.stream.GetLive())
	}
}

func (l logView) updateSize(update TeaLogSizeUpdate) (tea.Model, tea.Cmd) {
	l.width = update.Width
	l.height = update.Height
	return l, tea.WindowSize()
}
