package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/headless"
	"github.com/SpandanBG/logctrl/internal/logfeed"
	"github.com/SpandanBG/logctrl/internal/logview"
	"github.com/SpandanBG/logctrl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	tab   = tea.KeyMsg{Type: tea.KeyTab}   // the tab key
	esc   = tea.KeyMsg{Type: tea.KeyEsc}   // the esc key
	enter = tea.KeyMsg{Type: tea.KeyEnter} // the enter key
	ctrlL = tea.KeyMsg{Type: tea.KeyCtrlL} // ctrl+l
	ctrlF = tea.KeyMsg{Type: tea.KeyCtrlF} // ctrl+f
)

// topLine - the grey line along the top of a regex bar, as wide as the screen.
var topLine = strings.Repeat("─", 80)

// keys - the key message typing `text`.
func keys(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

// startUI - the UI on an 80x30 terminal over the real Executor, started the
// way the tea runtime starts it.
func startUI(t *testing.T) (tea.Model, *logfeed.Feed, *testutil.Runtime) {
	t.Helper()
	return startSizedUI(t, 80, 30)
}

// startSizedUI - the UI on a `width` x `height` terminal over the real
// Executor, started the way the tea runtime starts it.
func startSizedUI(t *testing.T, width, height int) (tea.Model, *logfeed.Feed, *testutil.Runtime) {
	t.Helper()
	return startRemappedUI(t, width, height, "[]")
}

// remapped - a user Keymap file moving every Action off its built-in key.
const remapped = `[
	{"action": "open_help", "key": "?"},
	{"action": "quit", "key": "x"},
	{"action": "open_prompt", "key": "p"},
	{"action": "close", "key": "ctrl+x"},
	{"action": "scroll_up", "key": "u"},
	{"action": "scroll_down", "key": "n"},
	{"action": "follow", "key": "end"},
	{"action": "open_search", "key": "s"},
	{"action": "close_search", "key": "ctrl+o"}
]`

// startRemappedUI - the UI on a `width` x `height` terminal with `userKeymap`
// as the user's Keymap file, over the real Executor, started the way the tea
// runtime starts it.
func startRemappedUI(t *testing.T, width, height int, userKeymap string) (tea.Model, *logfeed.Feed, *testutil.Runtime) {
	t.Helper()
	stream, feed := logfeed.New(t)
	runtime := testutil.NewRuntime(width, height)

	path := filepath.Join(t.TempDir(), "keymap.json")
	if err := os.WriteFile(path, []byte(userKeymap), 0o600); err != nil {
		t.Fatal(err)
	}
	keymap, err := config.LoadKeymap(path)
	if err != nil {
		t.Fatal(err)
	}

	model := tea.Model(newModel(headless.New(cmd.NewExecutor(stream, testutil.Colors())), keymap, testutil.Colors()))
	runtime.Run(model.Init())

	return runtime.Settle(model), feed, runtime
}

// typeKeys - presses each key in `text` in turn, as the user types it.
// Returns whether any of them quit the UI.
func typeKeys(model tea.Model, runtime *testutil.Runtime, text string) (tea.Model, bool) {
	quit := false
	for _, key := range text {
		var quits bool
		model, quits = press(model, runtime, keys(string(key)))
		quit = quit || quits
	}
	return model, quit
}

// arrive - writes `lines` to the Log Feed and settles `model` until they have
// all arrived.
func arrive(t *testing.T, model tea.Model, runtime *testutil.Runtime, feed *logfeed.Feed, lines ...string) tea.Model {
	t.Helper()
	feed.Write(lines...)
	return runtime.SettleUntil(t, model, feed.Arrived)
}

// logRows - how many rows tall the Log View is.
func logRows(model tea.Model) int {
	return len(logview.Rows(model.View()))
}

// press - sends `key` to the UI and settles it. Returns whether the UI quit.
func press(model tea.Model, runtime *testutil.Runtime, key tea.KeyMsg) (tea.Model, bool) {
	model, cmd := model.Update(key)
	cmd = once(cmd)
	quit := quits(cmd)
	runtime.Run(cmd)
	return runtime.Settle(model), quit
}

// once - `cmd` run at most once, every call returning its message, and so for
// each command in a batch, so checking it for Quit doesn't repeat its effects.
func once(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	var run sync.Once
	var msg tea.Msg
	return func() tea.Msg {
		run.Do(func() {
			msg = cmd()
			if batch, ok := msg.(tea.BatchMsg); ok {
				for i := range batch {
					batch[i] = once(batch[i])
				}
			}
		})
		return msg
	}
}

// quits - whether `cmd` (or any command in a batch) asks the program to quit.
// Commands still running after a short wait (e.g. cursor blinks) don't.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, each := range batch {
				if quits(each) {
					return true
				}
			}
			return false
		}
		_, ok := msg.(tea.QuitMsg)
		return ok
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

func Test_UIQuitsOnQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		keys("q"),
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyCtrlD},
	} {
		t.Run(key.String(), func(t *testing.T) {
			model, _, runtime := startUI(t)

			_, quit := press(model, runtime, key)

			testutil.Equal(t, true, quit)
		})
	}
}

func Test_UIFillsTerminalWithToolbarAndLogView(t *testing.T) {
	model, _, _ := startUI(t)

	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 border rows
	testutil.Equal(t, 27, logRows(model))
}

func Test_UITabOpensPromptAndShrinksLogView(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, tab)

	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 16 prompt rows - 2 border rows
	testutil.Equal(t, 11, logRows(model))
}

func Test_UISendsKeysToOpenPrompt(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	model, quit := press(model, runtime, keys("q"))
	model, _ = press(model, runtime, keys("uit now"))

	testutil.Equal(t, false, quit)
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "quit now"))
}

func Test_UIEscClosesPromptAndRestoresLogView(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("typed"))
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "typed"))
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, 27, logRows(model))
}

func Test_UIScrollKeysScrollLogViewWhilePromptClosed(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("j"))

	testutil.Equal(t, "ok 37", logview.Bottom(model.View()))
}

func Test_UIFollowKeyMakesLogViewFollowingWhilePromptClosed(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("G"))

	testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
}

func Test_UIScrollKeysAreTypedWhilePromptOpen(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, tab)
	before := logview.Bottom(model.View())
	model, _ = typeKeys(model, runtime, "jk")

	testutil.Equal(t, before, logview.Bottom(model.View())) // the Window has not moved
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "jk"))
}

func Test_UIHelpKeyOpensHelpInPlaceOfLogView(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, "a log line")

	model, _ = press(model, runtime, keys("h"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, true, strings.Contains(view, "Scroll the Log View or the Help down"))
	testutil.Equal(t, false, strings.Contains(view, "a log line"))
}

func Test_UIEscClosesHelpAndRestoresLogView(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, "a log line")

	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, esc)

	view := ansi.Strip(model.View())
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, false, strings.Contains(view, "Scroll the Log View or the Help down"))
	testutil.Equal(t, true, strings.Contains(view, "a log line"))
}

func Test_UIScrollKeysScrollHelpNotLogViewWhileHelpOpen(t *testing.T) {
	// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 Help rows
	model, feed, runtime := startSizedUI(t, 80, 5)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, keys("k"))
	view := ansi.Strip(model.View())
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, false, strings.Contains(view, "Open the Help"))
	testutil.Equal(t, true, strings.Contains(view, "Quit logctrl"))
	testutil.Equal(t, true, strings.Contains(view, "Open the Prompt"))
	testutil.Equal(t, "ok 39", logview.Bottom(model.View())) // the Log View did not scroll
}

func Test_UIIgnoresOtherKeysWhileHelpOpen(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		keys("q"),
		tab,
		keys("x"),
		keys("G"),
		keys("/"),
	} {
		t.Run(key.String(), func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model, _ = press(model, runtime, keys("k"))
			model, _ = press(model, runtime, keys("h"))
			helpView := model.View()

			model, quit := press(model, runtime, key)
			view := model.View()
			model, _ = press(model, runtime, esc)

			testutil.Equal(t, false, quit)
			testutil.Equal(t, helpView, view)                        // no panel opened, nothing typed
			testutil.Equal(t, "ok 38", logview.Bottom(model.View())) // the Window did not Follow
		})
	}
}

func Test_UIReservedKeysQuitWhileHelpOpen(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyCtrlD},
	} {
		t.Run(key.String(), func(t *testing.T) {
			model, _, runtime := startUI(t)
			model, _ = press(model, runtime, keys("h"))

			_, quit := press(model, runtime, key)

			testutil.Equal(t, true, quit)
		})
	}
}

func Test_UIHelpKeyIsTypedWhilePromptOpen(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, keys("i"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, true, strings.Contains(view, "hi"))
	testutil.Equal(t, false, strings.Contains(view, "Scroll the Log View or the Help down"))
}

func Test_UIReopenedHelpStartsAtTheTop(t *testing.T) {
	model, _, runtime := startSizedUI(t, 80, 5)

	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, esc)
	model, _ = press(model, runtime, keys("h"))

	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "Open the Help"))
}

func Test_UIQuitsOnRemappedQuitKeyNotTheBuiltInOne(t *testing.T) {
	model, _, runtime := startRemappedUI(t, 80, 30, remapped)

	model, quitOnBuiltIn := press(model, runtime, keys("q"))
	_, quitOnRemapped := press(model, runtime, keys("x"))

	testutil.Equal(t, false, quitOnBuiltIn)
	testutil.Equal(t, true, quitOnRemapped)
}

func Test_UIOpensHelpOnRemappedKeyNotTheBuiltInOne(t *testing.T) {
	model, _, runtime := startRemappedUI(t, 80, 30, remapped)

	model, _ = press(model, runtime, keys("h"))
	openedOnBuiltIn := strings.Contains(ansi.Strip(model.View()), "Open the Help")
	model, _ = press(model, runtime, keys("?"))
	openedOnRemapped := strings.Contains(ansi.Strip(model.View()), "Open the Help")

	testutil.Equal(t, false, openedOnBuiltIn)
	testutil.Equal(t, true, openedOnRemapped)
}

func Test_UIOpensAndClosesPromptOnRemappedKeysNotTheBuiltInOnes(t *testing.T) {
	model, _, runtime := startRemappedUI(t, 80, 30, remapped)

	model, _ = press(model, runtime, tab)
	openedOnBuiltIn := logRows(model) != 27
	model, _ = press(model, runtime, keys("p"))
	openedOnRemapped := logRows(model) != 27
	model, _ = press(model, runtime, esc)
	closedOnBuiltIn := logRows(model) == 27
	model, _ = press(model, runtime, tea.KeyMsg{Type: tea.KeyCtrlX})
	closedOnRemapped := logRows(model) == 27

	testutil.Equal(t, false, openedOnBuiltIn)
	testutil.Equal(t, true, openedOnRemapped)
	testutil.Equal(t, false, closedOnBuiltIn)
	testutil.Equal(t, true, closedOnRemapped)
}

func Test_UIClosesHelpOnRemappedKeyNotTheBuiltInOne(t *testing.T) {
	model, _, runtime := startRemappedUI(t, 80, 30, remapped)
	model, _ = press(model, runtime, keys("?"))

	model, _ = press(model, runtime, esc)
	closedOnBuiltIn := !strings.Contains(ansi.Strip(model.View()), "Open the Help")
	model, _ = press(model, runtime, tea.KeyMsg{Type: tea.KeyCtrlX})
	closedOnRemapped := !strings.Contains(ansi.Strip(model.View()), "Open the Help")

	testutil.Equal(t, false, closedOnBuiltIn)
	testutil.Equal(t, true, closedOnRemapped)
}

func Test_UIScrollsLogViewOnRemappedKeysNotTheBuiltInOnes(t *testing.T) {
	model, feed, runtime := startRemappedUI(t, 80, 30, remapped)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("u"))
	model, _ = press(model, runtime, keys("u"))
	model, _ = press(model, runtime, keys("u"))
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("n"))
	model, _ = press(model, runtime, keys("j"))

	testutil.Equal(t, "ok 37", logview.Bottom(model.View()))
}

func Test_UIFollowsOnRemappedKeyNotTheBuiltInOne(t *testing.T) {
	model, feed, runtime := startRemappedUI(t, 80, 30, remapped)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
	model, _ = press(model, runtime, keys("u"))

	model, _ = press(model, runtime, keys("G"))
	afterBuiltIn := logview.Bottom(model.View())
	model, _ = press(model, runtime, tea.KeyMsg{Type: tea.KeyEnd})

	testutil.Equal(t, "ok 38", afterBuiltIn)
	testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
}

func Test_UIScrollsHelpOnRemappedKeysNotTheBuiltInOnes(t *testing.T) {
	// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 Help rows
	model, feed, runtime := startRemappedUI(t, 80, 5, remapped)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
	model, _ = press(model, runtime, keys("?"))

	model, _ = press(model, runtime, keys("j"))
	scrolledOnBuiltIn := !strings.Contains(ansi.Strip(model.View()), "Open the Help")
	model, _ = press(model, runtime, keys("n"))
	model, _ = press(model, runtime, keys("n"))
	model, _ = press(model, runtime, keys("u"))
	view := ansi.Strip(model.View())
	model, _ = press(model, runtime, tea.KeyMsg{Type: tea.KeyCtrlX})

	testutil.Equal(t, false, scrolledOnBuiltIn)
	testutil.Equal(t, "ok 39", logview.Bottom(model.View())) // the Log View did not scroll
	testutil.Equal(t, false, strings.Contains(view, "Open the Help"))
	testutil.Equal(t, true, strings.Contains(view, "Quit logctrl"))
	testutil.Equal(t, true, strings.Contains(view, "Open the Prompt"))
}

func Test_UISearchKeyOpensSearchBarAndShrinksLogView(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("/"))
	before := logview.Bottom(model.View())
	model, quit := typeKeys(model, runtime, "q.*jk")

	testutil.Equal(t, false, quit)
	testutil.Equal(t, before, logview.Bottom(model.View())) // the Window has not moved
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 Search Bar rows - 2 border rows
	testutil.Equal(t, 25, logRows(model))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: q.*jk"))
}

func Test_UIEnterMakesSearchBarReadOnly(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("err"))
	model, _ = press(model, runtime, enter)
	model, _ = press(model, runtime, keys("k"))
	scrolled := logview.Bottom(model.View())
	model, _ = press(model, runtime, keys("x"))
	model, _ = press(model, runtime, keys("G"))

	testutil.Equal(t, "ok 38", scrolled)
	testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
	testutil.Equal(t, 25, logRows(model))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err"))
	testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "errk"))
}

func Test_UICloseSearchKeyClosesSearchBarAndRestoresLogView(t *testing.T) {
	for _, test := range []struct {
		name  string
		typed []tea.KeyMsg // after opening the Search Bar, before closing it
	}{
		{name: "while typing", typed: []tea.KeyMsg{keys("err")}},
		{name: "while read-only", typed: []tea.KeyMsg{keys("err"), enter}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model, _ = press(model, runtime, keys("/"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, ctrlL)
			closedView := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(closedView, "/: err"))
			testutil.Equal(t, 30, lipgloss.Height(model.View()))
			testutil.Equal(t, 27, logRows(model))
			testutil.Equal(t, "ok 38", logview.Bottom(model.View()))

			model, _ = press(model, runtime, keys("/"))
			testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "/: err"))
		})
	}
}

func Test_UIEscCancelsTypingInSearchBar(t *testing.T) {
	for _, test := range []struct {
		name     string
		typed    []tea.KeyMsg // after opening the Search Bar, before Esc
		gone     string       // what was typed since the regex was entered, not in the view after Esc
		height   int
		expected string // in the view after Esc, if anything
	}{
		{name: "first edit closes the Search Bar", typed: []tea.KeyMsg{keys("err")}, gone: "/: err", height: 27},
		{name: "re-edit restores the entered regex", typed: []tea.KeyMsg{keys("err"), enter, keys("/"), keys("or")}, gone: "/: erro", height: 25, expected: "/: err"},
		{name: "re-edit after a bad regex shows its error again", typed: []tea.KeyMsg{keys("err("), enter, keys("/"), keys(")")}, gone: "/: err()", height: 25, expected: "/: error parsing regexp: missing closing ): `err(`"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model, _ = press(model, runtime, keys("/"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, esc)
			view := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(view, test.gone))
			testutil.Equal(t, true, strings.Contains(view, test.expected))
			testutil.Equal(t, test.height, logRows(model))
			testutil.Equal(t, "ok 38", logview.Bottom(model.View()))
		})
	}
}

// withMatches - 40 Lines, two of them with a Match for "err" and one for
// "error", all in the Log View.
func withMatches() []string {
	lines := logfeed.Numbered(40, 30)
	lines[35] = "error 35"
	return lines
}

func Test_UISearchStartsWhenRegexIsEntered(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, withMatches()...)
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("err"))

	testutil.Equal(t, "[]", logview.Highlighted(model.View()))

	model, _ = press(model, runtime, enter)
	testutil.Equal(t, "[err err]", logview.Highlighted(model.View()))

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("or"))
	model, _ = press(model, runtime, enter)
	testutil.Equal(t, "[error]", logview.Highlighted(model.View()))
}

func Test_UICloseSearchKeyEndsSearch(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, withMatches()...)
	model = enterSearch(model, runtime, "err")

	model, _ = press(model, runtime, ctrlL)

	testutil.Equal(t, "[]", logview.Highlighted(model.View()))
}

func Test_UICloseSearchKeyFollowsOnceRegexIsEntered(t *testing.T) {
	for _, test := range []struct {
		name  string
		regex string
		typed []tea.KeyMsg // after entering the regex and scrolling, before closing the Search Bar
	}{
		{name: "while read-only", regex: "err"},
		{name: "while editing", regex: "err", typed: []tea.KeyMsg{keys("/"), keys("or")}},
		{name: "after a bad regex", regex: "err("},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model = enterSearch(model, runtime, test.regex)
			model, _ = press(model, runtime, keys("k"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, ctrlL)

			testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
		})
	}
}

func Test_UIClosingSearchBarBeforeRegexIsEnteredKeepsWindow(t *testing.T) {
	for _, test := range []struct {
		name  string
		close tea.KeyMsg
	}{
		{name: "close search key", close: ctrlL},
		{name: "esc", close: esc},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model, _ = press(model, runtime, keys("k"))
			model, _ = press(model, runtime, keys("/"))
			model, _ = press(model, runtime, keys("err"))

			model, _ = press(model, runtime, test.close)

			testutil.Equal(t, "ok 38", logview.Bottom(model.View()))
		})
	}
}

func Test_UIEscKeepsEnteredSearch(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, withMatches()...)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("or"))

	model, _ = press(model, runtime, esc)

	testutil.Equal(t, "[err err]", logview.Highlighted(model.View()))
}

func Test_UIBadRegexShowsItsErrorInRedInSearchBar(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model = enterSearch(model, runtime, "err(")
	model, _ = press(model, runtime, keys("k"))

	testutil.Equal(t, true, strings.Contains(model.View(),
		"\x1b[31merror parsing regexp: missing closing ): `err(`\x1b[0m"))
	testutil.Equal(t, "ok 38", logview.Bottom(model.View()))
}

func Test_UILogViewRedrawsWhenSearchStartsOrEnds(t *testing.T) {
	for _, test := range []struct {
		name     string
		regexes  []string // entered in turn
		closing  bool     // close the Search Bar after entering `regexes`
		expected string   // the Matches highlighted
	}{
		{name: "Search starts", regexes: []string{"err"}, expected: "[err err]"},
		{name: "bad regex ends the Search", regexes: []string{"err", "err("}, expected: "[]"},
		{name: "Search Bar closes", regexes: []string{"err"}, closing: true, expected: "[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, withMatches()...)

			for _, regex := range test.regexes {
				model = enterSearch(model, runtime, regex)
			}
			if test.closing {
				model, _ = press(model, runtime, ctrlL)
			}

			testutil.Equal(t, test.expected, logview.Highlighted(model.View()))
		})
	}
}

// enterSearch - opens the Search Bar and enters `regex` in it.
func enterSearch(model tea.Model, runtime *testutil.Runtime, regex string) tea.Model {
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys(regex))
	model, _ = press(model, runtime, enter)
	return model
}

func Test_UIHelpOpensAboveReadOnlySearchBar(t *testing.T) {
	model, _, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")

	model, _ = press(model, runtime, keys("h"))
	helpView := ansi.Strip(model.View())
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, true, strings.Contains(helpView, "Scroll the Log View or the Help down"))
	testutil.Equal(t, true, strings.HasSuffix(strings.TrimSpace(helpView), "/: err"))
	testutil.Equal(t, 30, lipgloss.Height(helpView))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err"))
}

func Test_UIPromptOpensBelowReadOnlySearchBar(t *testing.T) {
	model, _, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("cmd"))
	promptView := ansi.Strip(model.View())
	promptHeight := logRows(model)
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, 30, lipgloss.Height(promptView))
	// 30 rows - 1 toolbar row - 2 Search Bar rows - 16 prompt rows - 2 border rows
	testutil.Equal(t, 9, promptHeight)
	testutil.Equal(t, true, strings.Index(promptView, "/: err") < strings.Index(promptView, "cmd"))
	testutil.Equal(t, 25, logRows(model))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err"))
}

func Test_UIReservedKeysQuitWhileTypingInSearchBarOrFilterBar(t *testing.T) {
	for _, open := range []string{"/", "f"} {
		for _, key := range []tea.KeyMsg{
			{Type: tea.KeyCtrlC},
			{Type: tea.KeyCtrlD},
		} {
			t.Run(open+" "+key.String(), func(t *testing.T) {
				model, _, runtime := startUI(t)
				model, _ = press(model, runtime, keys(open))

				_, quit := press(model, runtime, key)

				testutil.Equal(t, true, quit)
			})
		}
	}
}

func Test_UISearchKeyIsTypedWhilePromptOpen(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	for _, key := range []string{"a", "/", "b"} {
		model, _ = press(model, runtime, keys(key))
	}

	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "a/b"))
	// the Search Bar stayed closed: 30 rows - 1 toolbar row - 16 prompt rows - 2 border rows
	testutil.Equal(t, 11, logRows(model))
}

func Test_UIOpensAndClosesSearchBarOnRemappedKeysNotTheBuiltInOnes(t *testing.T) {
	for _, test := range []struct {
		name     string
		builtIn  tea.KeyMsg
		remapped tea.KeyMsg
	}{
		{name: "close_search", builtIn: ctrlL, remapped: tea.KeyMsg{Type: tea.KeyCtrlO}},
		{name: "close", builtIn: esc, remapped: tea.KeyMsg{Type: tea.KeyCtrlX}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, _, runtime := startRemappedUI(t, 80, 30, remapped)

			model, _ = press(model, runtime, keys("/"))
			openedOnBuiltIn := logRows(model) != 27
			model, _ = press(model, runtime, keys("s"))
			openedOnRemapped := logRows(model) != 27
			model, _ = press(model, runtime, keys("err"))
			model, _ = press(model, runtime, test.builtIn)
			closedOnBuiltIn := logRows(model) == 27
			model, _ = press(model, runtime, test.remapped)
			closedOnRemapped := logRows(model) == 27

			testutil.Equal(t, false, openedOnBuiltIn)
			testutil.Equal(t, true, openedOnRemapped)
			testutil.Equal(t, false, closedOnBuiltIn)
			testutil.Equal(t, true, closedOnRemapped)
		})
	}
}

func Test_UIFilterKeyOpensFilterBarAndShrinksLogView(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)

	model, _ = press(model, runtime, keys("f"))
	before := logview.Bottom(model.View())
	model, quit := typeKeys(model, runtime, "q.*jk/")

	testutil.Equal(t, false, quit)
	testutil.Equal(t, before, logview.Bottom(model.View())) // the Window has not moved
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 Filter Bar rows - 2 border rows
	testutil.Equal(t, 25, logRows(model))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "f: q.*jk/"))
}

func Test_UIEnterMakesFilterBarReadOnlyAndFiltersWithoutSearching(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 20, 30)...)
	model, _ = press(model, runtime, keys("k")) // Pins the Window

	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys("ok"))
	model, _ = press(model, runtime, enter)
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("x"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, false, strings.Contains(fmt.Sprint(logview.ShownLines(model.View())), "err"))
	testutil.Equal(t, "[]", logview.Highlighted(model.View()))
	testutil.Equal(t, "ok 37", logview.Bottom(model.View())) // scrolled one shown Line, never Following
	testutil.Equal(t, 25, logRows(model))
	testutil.Equal(t, true, strings.Contains(view, "f: ok"))
	testutil.Equal(t, false, strings.Contains(view, "okk"))
	testutil.Equal(t, false, strings.Contains(view, "/: ok"))
}

func Test_UICloseFilterKeyClosesFilterBarEndingFilterAndRestoresLogView(t *testing.T) {
	for _, test := range []struct {
		name  string
		typed []tea.KeyMsg // after opening the Filter Bar, before closing it
	}{
		{name: "while typing", typed: []tea.KeyMsg{keys("ok")}},
		{name: "while read-only", typed: []tea.KeyMsg{keys("ok"), enter}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 20, 30)...)
			model, _ = press(model, runtime, keys("k")) // Pins the Window
			model, _ = press(model, runtime, keys("f"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, ctrlF)
			closedView := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(closedView, "f: ok"))
			testutil.Equal(t, 30, lipgloss.Height(model.View()))
			testutil.Equal(t, 27, logRows(model))
			testutil.Equal(t, true, slices.Contains(logview.ShownLines(model.View()), "err 30")) // the Filter ended
			testutil.Equal(t, "ok 37", logview.Bottom(model.View()))                             // scrolled, never Following

			model, _ = press(model, runtime, keys("f"))
			testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "f: ok"))
		})
	}
}

func Test_UIBadRegexShowsItsErrorInRedInFilterBar(t *testing.T) {
	model, _, runtime := startUI(t)

	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys("err("))
	model, _ = press(model, runtime, enter)

	testutil.Equal(t, true, strings.Contains(model.View(),
		"\x1b[31merror parsing regexp: missing closing ): `err(`\x1b[0m"))
}

func Test_UILogViewRedrawsWhenFilterStartsOrEnds(t *testing.T) {
	for _, test := range []struct {
		name     string
		regexes  []string // entered in turn
		closing  bool     // close the Filter Bar after entering `regexes`
		expected string   // the bottom Line shown
	}{
		{name: "Filter starts", regexes: []string{"err"}, expected: "err 30"},
		{name: "bad regex ends the Filter", regexes: []string{"err", "err("}, expected: "ok 39"},
		{name: "Filter Bar closes", regexes: []string{"err"}, closing: true, expected: "ok 39"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 30)...)

			for _, regex := range test.regexes {
				model = enterFilter(model, runtime, regex)
			}
			if test.closing {
				model, _ = press(model, runtime, ctrlF)
			}

			testutil.Equal(t, test.expected, logview.Bottom(model.View()))
		})
	}
}

// enterFilter - opens the Filter Bar and enters `regex` in it.
func enterFilter(model tea.Model, runtime *testutil.Runtime, regex string) tea.Model {
	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys(regex))
	model, _ = press(model, runtime, enter)
	return model
}

func Test_UIEscCancelsTypingInFilterBar(t *testing.T) {
	for _, test := range []struct {
		name     string
		typed    []tea.KeyMsg // after opening the Filter Bar, before Esc
		height   int
		expected string // in the view after Esc, if anything
	}{
		{name: "first edit closes the Filter Bar", typed: []tea.KeyMsg{keys("ok")}, height: 27},
		{name: "re-edit restores the entered regex", typed: []tea.KeyMsg{keys("ok"), enter, keys("f"), keys("ay")}, height: 25, expected: "f: ok"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40)...)
			model, _ = press(model, runtime, keys("f"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, esc)
			view := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(view, "f: okay"))
			testutil.Equal(t, test.expected != "", strings.Contains(view, "f: ok"))
			testutil.Equal(t, test.height, logRows(model))
			testutil.Equal(t, "ok 38", logview.Bottom(model.View()))
		})
	}
}

func Test_UIFilterBarSitsAboveSearchBar(t *testing.T) {
	model, _, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys("warn"))
	model, _ = press(model, runtime, enter)

	view := ansi.Strip(model.View())
	rows := logRows(model)
	model, _ = press(model, runtime, keys("h"))
	helpView := ansi.Strip(model.View())

	testutil.Equal(t, 30, lipgloss.Height(view))
	// 30 rows - 1 toolbar row - 2 Filter Bar rows - 2 Search Bar rows - 2 border rows
	testutil.Equal(t, 23, rows)
	testutil.Equal(t, fmt.Sprint([]string{topLine, "f: warn", topLine, "/: err"}), fmt.Sprint(lastRows(view, 4)))
	testutil.Equal(t, 30, lipgloss.Height(helpView))
	testutil.Equal(t, fmt.Sprint([]string{topLine, "f: warn", topLine, "/: err"}), fmt.Sprint(lastRows(helpView, 4)))
}

// lastRows - the last `n` rows of `view`, without their padding.
func lastRows(view string, n int) []string {
	rows := strings.Split(view, "\n")
	rows = rows[len(rows)-n:]
	for i := range rows {
		rows[i] = strings.TrimSpace(rows[i])
	}
	return rows
}

func Test_UITypingInOneRegexBarNeverClosesTheOther(t *testing.T) {
	for _, test := range []struct {
		name     string
		open     tea.KeyMsg // opens the bar typed in, after the other is entered
		other    string     // the regex entered in the other bar
		key      tea.KeyMsg // the other bar's close key, then its open key, typed
		expected []string   // the rows of both bars
	}{
		{name: "typing in Filter Bar", open: keys("f"), other: "/err", key: ctrlL, expected: []string{topLine, "f: a/b", topLine, "/: err"}},
		{name: "typing in Search Bar", open: keys("/"), other: "fwarn", key: ctrlF, expected: []string{topLine, "f: warn", topLine, "/: afb"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, _, runtime := startUI(t)
			model, _ = press(model, runtime, keys(test.other[:1]))
			model, _ = press(model, runtime, keys(test.other[1:]))
			model, _ = press(model, runtime, enter)
			model, _ = press(model, runtime, test.open)

			model, _ = press(model, runtime, keys("a"))
			model, _ = press(model, runtime, test.key)
			model, _ = press(model, runtime, keys(test.other[:1]))
			model, _ = press(model, runtime, keys("b"))

			testutil.Equal(t, fmt.Sprint(test.expected), fmt.Sprint(lastRows(ansi.Strip(model.View()), 4)))
			testutil.Equal(t, 23, logRows(model))
		})
	}
}

func Test_UIReservedKeysDoNotQuitWhilePromptOpen(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyCtrlD},
	} {
		t.Run(key.String(), func(t *testing.T) {
			model, _, runtime := startUI(t)
			model, _ = press(model, runtime, tab)

			_, quit := press(model, runtime, key)

			testutil.Equal(t, false, quit)
		})
	}
}

var (
	up   = tea.KeyMsg{Type: tea.KeyUp}   // the up arrow
	down = tea.KeyMsg{Type: tea.KeyDown} // the down arrow
)

func Test_UIStepKeysStepBetweenLinesWithMatchOnlyWhileRegexEnteredAndReadOnly(t *testing.T) {
	for _, test := range []struct {
		name     string
		before   []tea.KeyMsg // pressed after entering a Search, unless `noSearch`; closed by esc
		noSearch bool
		expected string // the Current Line after the steps
	}{
		{name: "regex entered", expected: "err 20"},
		{name: "no Search", noSearch: true},
		{name: "typing in the Search Bar", before: []tea.KeyMsg{keys("/")}},
		{name: "Help open", before: []tea.KeyMsg{keys("h")}},
		{name: "Prompt open", before: []tea.KeyMsg{tab}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 10, 20, 30)...)
			if !test.noSearch {
				model = enterSearch(model, runtime, "err")
			}
			for _, key := range test.before {
				model, _ = press(model, runtime, key)
			}

			for _, key := range []tea.KeyMsg{up, up, down, up} { // to err 30, 20, 30, then 20
				model, _ = press(model, runtime, key)
			}
			if len(test.before) > 0 {
				model, _ = press(model, runtime, esc)
			}

			testutil.Equal(t, test.expected, logview.Marked(model.View()))
		})
	}
}

func Test_UILogViewRedrawsAfterStep(t *testing.T) {
	for _, key := range []tea.KeyMsg{up, down} {
		t.Run(key.String(), func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 30)...)
			model = enterSearch(model, runtime, "err")

			model, _ = press(model, runtime, key)

			testutil.Equal(t, "err 30", logview.Marked(model.View()))
		})
	}
}

func Test_UIEnterWithNewRegexMakesWindowFollowing(t *testing.T) {
	for _, regex := range []string{"err", "err("} {
		t.Run(regex, func(t *testing.T) {
			model, feed, runtime := startUI(t)
			model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 30)...)
			model, _ = press(model, runtime, keys("k"))

			model = enterSearch(model, runtime, regex)

			testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
		})
	}
}

func Test_UIEnterWithSameRegexChangesNothing(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 10, 20)...)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, up) // Pins the Window on the Current Line
	pinned := logview.Bottom(model.View())

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, enter)

	testutil.Equal(t, "err 20", logview.Marked(model.View()))
	testutil.Equal(t, pinned, logview.Bottom(model.View()))
}

func Test_UIFollowKeyDropsCurrentLineKeepingSearch(t *testing.T) {
	model, feed, runtime := startUI(t)
	model = arrive(t, model, runtime, feed, logfeed.Numbered(40, 10, 20)...)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, up)

	model, _ = press(model, runtime, keys("G"))

	testutil.Equal(t, "", logview.Marked(model.View()))
	testutil.Equal(t, "[err]", logview.Highlighted(model.View()))
	testutil.Equal(t, "ok 39", logview.Bottom(model.View()))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err"))
}
