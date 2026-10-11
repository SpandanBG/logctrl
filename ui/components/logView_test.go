package components_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/internal/fakestream"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// startLogView - a Log View under a toolbar on an 80x30 terminal, started the
// way the tea runtime starts it.
func startLogView(t *testing.T) (tea.Model, *fakestream.Stream, *testutil.Runtime) {
	t.Helper()
	stream := fakestream.New()
	runtime := testutil.NewRuntime(80, 30)

	logView := components.NewLogView(ui.SizeRatio(1), ui.SizeModifier(-1), stream)
	runtime.Run(logView.Init())

	return runtime.Settle(logView), stream, runtime
}

func Test_LogViewSetsWindowHeightToItsInnerHeight(t *testing.T) {
	_, stream, _ := startLogView(t)

	// 30 rows - 1 toolbar row - 2 border rows
	testutil.Equal(t, 27, stream.Height())
}

func Test_LogViewShowsWindowWhenNewLinesArrive(t *testing.T) {
	logView, stream, runtime := startLogView(t)

	stream.Notify(t, "first line\nsecond line")
	logView = runtime.Settle(logView)

	view := ansi.Strip(logView.View())
	testutil.Equal(t, true, strings.Contains(view, "first line"))
	testutil.Equal(t, true, strings.Contains(view, "second line"))
}

func Test_LogViewKeepsShowingWindowAsMoreLinesArrive(t *testing.T) {
	logView, stream, runtime := startLogView(t)

	stream.Notify(t, "first line")
	logView = runtime.Settle(logView)
	stream.Notify(t, "second line")
	logView = runtime.Settle(logView)

	view := ansi.Strip(logView.View())
	testutil.Equal(t, false, strings.Contains(view, "first line"))
	testutil.Equal(t, true, strings.Contains(view, "second line"))
}

func Test_LogViewResizesWindowWhenItsSizeChanges(t *testing.T) {
	logView, stream, runtime := startLogView(t)

	logView, cmd := logView.Update(components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-17),
	})
	runtime.Run(cmd)
	runtime.Settle(logView)

	// 30 rows - 17 (toolbar + prompt) - 2 border rows
	testutil.Equal(t, 11, stream.Height())
}

func Test_LogViewScrollMovesWindowByItsLines(t *testing.T) {
	for _, test := range []struct {
		name     string
		scroll   components.TeaScroll
		expected int
	}{
		{name: "up", scroll: -1, expected: -1},
		{name: "down", scroll: 1, expected: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			logView, stream, runtime := startLogView(t)

			logView, cmd := logView.Update(test.scroll)
			runtime.Run(cmd)
			runtime.Settle(logView)

			testutil.Equal(t, fmt.Sprint([]int{test.expected}), fmt.Sprint(stream.Scrolls()))
		})
	}
}

func Test_LogViewShowsScrolledWindowWithoutWaitingForNewLines(t *testing.T) {
	logView, stream, runtime := startLogView(t)
	stream.Notify(t, "newest line")
	logView = runtime.Settle(logView)

	stream.SetWindow("older line")
	logView, cmd := logView.Update(components.TeaScroll(-1))
	runtime.Run(cmd)
	logView = runtime.Settle(logView)

	view := ansi.Strip(logView.View())
	testutil.Equal(t, true, strings.Contains(view, "older line"))
	testutil.Equal(t, false, strings.Contains(view, "newest line"))
}
