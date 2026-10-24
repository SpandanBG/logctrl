package components

import (
	"github.com/SpandanBG/logctrl/cmd"
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

// TeaScroll - Scroll by this many lines: negative is up (older), positive is
// down (newer).
type TeaScroll int

// TeaFollow - make the Window Following.
type TeaFollow struct{}

// TeaLogRefresh - show the Window again, e.g. after a Search starts or ends.
type TeaLogRefresh struct{}

// ----- Private tea.Msg
type teaLogCmd struct {
	shown  string // the Window's shown text
	gutter string // the Gutter cells of its Lines
}

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

type logView struct {
	width   ui.SizeI       // width modifier
	height  ui.SizeI       // height modifier
	view    viewport.Model // holds the viewport
	gutter  gutterView     // the Gutter, left of the viewport
	stream  cmd.Stream     // log feed stream to be displayed
	ready   bool           // if `true` the viewport is ready to render
	nextLog chan bool      // notification channel from stream
}

// NewLogView - the Log View over `stream`, its Gutter styled by `renderer`.
func NewLogView(width, height ui.SizeI, stream cmd.Stream, renderer *lipgloss.Renderer) tea.Model {
	nextLog := make(chan bool)

	stream.SetBufferSize(1)
	stream.Start(nextLog)

	return logView{
		width:   width,
		height:  height,
		stream:  stream,
		nextLog: nextLog,
		gutter:  newGutterView(renderer),
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
	case tea.WindowSizeMsg:
		return l.updateViewSize(msg)
	case teaLogCmd:
		return l.refreshView(msg)
	case TeaLogSizeUpdate:
		return l.updateSize(msg)
	case TeaScroll:
		return l.scroll(int(msg))
	case TeaFollow:
		return l.follow()
	case TeaLogRefresh:
		return l.show(), nil
	}
	return l, nil
}

func (l logView) View() string {
	return logViewStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Top, l.gutter.View(), l.view.View()),
	)
}

// ------------------------- Private
// scroll - scrolls the window by `n` lines and shows it right away, without
// waiting for new lines to arrive.
func (l logView) scroll(n int) (tea.Model, tea.Cmd) {
	l.stream.MoveBy(n)
	return l.show(), nil
}

// follow - makes the window Following and shows it right away, without waiting
// for new lines to arrive.
func (l logView) follow() (tea.Model, tea.Cmd) {
	l.stream.MoveToEnd()
	return l.show(), nil
}

func (l logView) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// get relative size of view
	size = ui.ModifySize(size, l.width, l.height)
	w := size.Width - logViewStyle.GetHorizontalFrameSize() - reader.GutterWidth
	h := size.Height - logViewStyle.GetVerticalFrameSize()
	l.gutter.height = h

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

func (l logView) refreshView(window teaLogCmd) (tea.Model, tea.Cmd) {
	l = l.display(window)
	return l, l.fetchLog()
}

func (l logView) fetchLog() tea.Cmd {
	return func() tea.Msg {
		<-l.nextLog
		return l.window()
	}
}

// window - the Window's shown text and Gutter cells, from one read.
func (l logView) window() teaLogCmd {
	shown, gutter := l.stream.GetWindow()
	return teaLogCmd{shown: shown, gutter: gutter}
}

// show - shows the Window as it is now, without waiting for new lines.
func (l logView) show() logView {
	return l.display(l.window())
}

// display - shows `window`'s shown text in the viewport and its Gutter cells
// beside it.
func (l logView) display(window teaLogCmd) logView {
	l.view.SetContent(window.shown)
	l.gutter = l.gutter.setCells(window.gutter)
	return l
}

func (l logView) updateSize(update TeaLogSizeUpdate) (tea.Model, tea.Cmd) {
	l.width = update.Width
	l.height = update.Height
	return l, tea.WindowSize()
}
