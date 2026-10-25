package components

import (
	"strings"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/headless"
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
type teaLogCmd struct {
	shown  string // the Window's shown text
	gutter string // the Gutter cells of its Lines
}

var (
	logViewStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
)

// logView - the Log View: the Window of a headless.Logs in a viewport, with
// its Gutter beside it, drawn on each notice.
type logView struct {
	width  ui.SizeI       // width modifier
	height ui.SizeI       // height modifier
	view   viewport.Model // holds the viewport
	gutter gutterView     // the Gutter, left of the viewport
	logs   *headless.Logs // the Window to be displayed
	ready  bool           // if `true` the viewport is ready to render
}

// NewLogView - the Log View showing the Window of `logs`, its Gutter styled
// by `renderer`.
func NewLogView(width, height ui.SizeI, logs *headless.Logs, renderer *lipgloss.Renderer) tea.Model {
	return logView{
		width:  width,
		height: height,
		logs:   logs,
		gutter: newGutterView(renderer),
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
	}
	return l, nil
}

func (l logView) View() string {
	return logViewStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Top, l.gutter.View(), l.view.View()),
	)
}

// ------------------------- Private

// updateViewSize - fits the viewport and the Gutter into the share of the
// terminal `size` the Log View is given, making the Window as tall.
func (l logView) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// get relative size of view
	size = ui.ModifySize(size, l.width, l.height)
	w := size.Width - logViewStyle.GetHorizontalFrameSize() - headless.GutterWidth
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

	// make the Window as tall as the viewport; resizing never fails
	l.logs.Update(cmd.Resize{Height: h})

	return l, nil
}

func (l logView) refreshView(window teaLogCmd) (tea.Model, tea.Cmd) {
	l = l.display(window)
	return l, l.fetchLog()
}

// fetchLog - the Window's shown text and Gutter cells, from one read, once it
// may have changed.
func (l logView) fetchLog() tea.Cmd {
	return func() tea.Msg {
		if _, open := <-l.logs.Changed(); !open {
			return nil // the Logs are closed: nothing more to show
		}
		shown, gutter := l.logs.Window()
		return teaLogCmd{shown: strings.Join(shown, "\n"), gutter: strings.Join(gutter, "\n")}
	}
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
