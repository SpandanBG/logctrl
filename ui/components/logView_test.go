package components_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/headless"
	"github.com/SpandanBG/logctrl/internal/logfeed"
	"github.com/SpandanBG/logctrl/internal/logview"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// startLogView - a Log View under a toolbar on an 80x30 terminal over the
// real Executor, started the way the tea runtime starts it.
func startLogView(t *testing.T) (tea.Model, *headless.Logs, *logfeed.Feed, *testutil.Runtime) {
	t.Helper()
	stream, feed := logfeed.New(t)
	logs := headless.New(cmd.NewExecutor(stream, testutil.Colors()))
	runtime := testutil.NewRuntime(80, 30)

	logView := components.NewLogView(ui.SizeRatio(1), ui.SizeModifier(-1), logs, testutil.Colors())
	runtime.Run(logView.Init())

	return runtime.Settle(logView), logs, feed, runtime
}

// arrive - writes `lines` to the Log Feed and settles `model` until they have
// all arrived.
func arrive(t *testing.T, model tea.Model, runtime *testutil.Runtime, feed *logfeed.Feed, lines ...string) tea.Model {
	t.Helper()
	feed.Write(lines...)
	return runtime.SettleUntil(t, model, feed.Arrived)
}

// update - hands `msg` to `model` and settles it.
func update(model tea.Model, runtime *testutil.Runtime, msg tea.Msg) tea.Model {
	model, cmd := model.Update(msg)
	runtime.Run(cmd)
	return runtime.Settle(model)
}

// change - hands `msg` to `logs`, as the layout does, and settles `model`.
func change(t *testing.T, model tea.Model, runtime *testutil.Runtime, logs *headless.Logs, msg any) tea.Model {
	t.Helper()
	if err := logs.Update(msg); err != nil {
		t.Fatal(err)
	}
	return runtime.Settle(model)
}

func Test_LogViewSetsWindowHeightToItsInnerHeight(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)

	logView = arrive(t, logView, runtime, feed, logfeed.Numbered(40)...)

	// 30 rows - 1 toolbar row - 2 border rows
	testutil.Equal(t, fmt.Sprint(logfeed.Numbered(40)[13:]), fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewShowsWindowWhenNewLinesArrive(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)

	logView = arrive(t, logView, runtime, feed, "first line", "second line")

	testutil.Equal(t, "[first line second line]", fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewKeepsShowingWindowAsMoreLinesArrive(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)

	logView = arrive(t, logView, runtime, feed, "first line")
	logView = arrive(t, logView, runtime, feed, "second line")

	testutil.Equal(t, "[first line second line]", fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewResizesWindowWhenItsSizeChanges(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)
	logView = arrive(t, logView, runtime, feed, logfeed.Numbered(40)...)

	logView = update(logView, runtime, components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-17),
	})
	logView = arrive(t, logView, runtime, feed, "ok 40")

	// 30 rows - 17 (toolbar + prompt) - 2 border rows
	testutil.Equal(t, 11, len(logview.Rows(logView.View())))
	testutil.Equal(t, fmt.Sprint(logfeed.Numbered(41)[30:]), fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewShowsShrunkWindowWithoutWaitingForNewLines(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)
	logView = arrive(t, logView, runtime, feed, logfeed.Numbered(40)...)

	logView = update(logView, runtime, components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-17),
	})

	// 30 rows - 17 (toolbar + prompt) - 2 border rows
	testutil.Equal(t, 11, len(logview.Rows(logView.View())))
	testutil.Equal(t, fmt.Sprint(logfeed.Numbered(40)[29:]), fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewShowsScrolledWindowWithoutWaitingForNewLines(t *testing.T) {
	logView, logs, feed, runtime := startLogView(t)
	logView = arrive(t, logView, runtime, feed, "older line")
	logView = update(logView, runtime, components.TeaLogSizeUpdate{
		Width:  ui.SizeRatio(1),
		Height: ui.SizeModifier(-27), // 30 rows - 27 - 2 border rows: one Line
	})
	logView = arrive(t, logView, runtime, feed, "newest line")

	logView = change(t, logView, runtime, logs, cmd.Scroll{N: -1})

	testutil.Equal(t, "[older line]", fmt.Sprint(logview.ShownLines(logView.View())))
}

func Test_LogViewFollowShowsNewestLinesWithoutWaitingForNewLines(t *testing.T) {
	logView, logs, feed, runtime := startLogView(t)
	logView = arrive(t, logView, runtime, feed, logfeed.Numbered(40)...)
	logView = change(t, logView, runtime, logs, cmd.Scroll{N: -1})
	testutil.Equal(t, "ok 38", logview.Bottom(logView.View()))

	logView = change(t, logView, runtime, logs, cmd.Follow{})

	testutil.Equal(t, "ok 39", logview.Bottom(logView.View()))
}

// stepTo - makes the Line matching `regex` the Current Line, as stepping to
// it does.
func stepTo(t *testing.T, model tea.Model, runtime *testutil.Runtime, logs *headless.Logs, regex string) tea.Model {
	t.Helper()
	model = change(t, model, runtime, logs, cmd.SearchFor{Regex: regex})
	return change(t, model, runtime, logs, cmd.PreviousMatch{})
}

func Test_LogViewShowsEachLinesGutterBesideIt(t *testing.T) {
	logView, logs, feed, runtime := startLogView(t)
	logView = arrive(t, logView, runtime, feed, logfeed.Numbered(40)...) // fills the Window, Pinned or not

	logView = stepTo(t, logView, runtime, logs, "39")

	rows := strings.Split(ansi.Strip(logView.View()), "\n")
	testutil.Equal(t, 30-1, len(rows))
	testutil.Equal(t, true, strings.HasPrefix(rows[26], "│  ok 38"))
	testutil.Equal(t, true, strings.HasPrefix(rows[27], "│ >ok 39"))
	for i, row := range rows {
		if ansi.StringWidth(row) != 80 {
			t.Errorf("row %d is %d wide, not 80: %q", i, ansi.StringWidth(row), row)
		}
	}
}

func Test_LogViewShadesItsGutterAllTheWayDown(t *testing.T) {
	logView, _, feed, runtime := startLogView(t)

	logView = arrive(t, logView, runtime, feed, "a log line")

	rows := strings.Split(logView.View(), "\n")
	testutil.Equal(t, true, strings.HasPrefix(rows[1], "│\x1b[40m  \x1b[0ma log line"))
	for i := 2; i <= 27; i++ { // the rows below the last Line
		if !strings.HasPrefix(rows[i], "│\x1b[40m  \x1b[0m") {
			t.Errorf("row %d has no shaded Gutter: %q", i, rows[i])
		}
	}
}
