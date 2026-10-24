package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/internal/fakestream"
	"github.com/SpandanBG/logctrl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	tab   = tea.KeyMsg{Type: tea.KeyTab}
	esc   = tea.KeyMsg{Type: tea.KeyEsc}
	enter = tea.KeyMsg{Type: tea.KeyEnter}
	ctrlL = tea.KeyMsg{Type: tea.KeyCtrlL}
	ctrlF = tea.KeyMsg{Type: tea.KeyCtrlF}
)

func keys(text string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)}
}

// startUI - the UI on an 80x30 terminal, started the way the tea runtime
// starts it.
func startUI(t *testing.T) (tea.Model, *fakestream.Stream, *testutil.Runtime) {
	t.Helper()
	return startSizedUI(t, 80, 30)
}

// startSizedUI - the UI on a `width` x `height` terminal, started the way the
// tea runtime starts it.
func startSizedUI(t *testing.T, width, height int) (tea.Model, *fakestream.Stream, *testutil.Runtime) {
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
// as the user's Keymap file, started the way the tea runtime starts it.
func startRemappedUI(t *testing.T, width, height int, userKeymap string) (tea.Model, *fakestream.Stream, *testutil.Runtime) {
	t.Helper()
	stream := fakestream.New()
	runtime := testutil.NewRuntime(width, height)

	path := filepath.Join(t.TempDir(), "keymap.json")
	if err := os.WriteFile(path, []byte(userKeymap), 0o600); err != nil {
		t.Fatal(err)
	}
	keymap, err := config.LoadKeymap(path)
	if err != nil {
		t.Fatal(err)
	}

	model := tea.Model(newModel(stream, keymap, testutil.Colors()))
	runtime.Run(model.Init())

	return runtime.Settle(model), stream, runtime
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
	model, stream, _ := startUI(t)

	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 border rows
	testutil.Equal(t, 27, stream.Height())
}

func Test_UITabOpensPromptAndShrinksLogView(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, tab)

	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 16 prompt rows - 2 border rows
	testutil.Equal(t, 11, stream.Height())
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
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("typed"))
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "typed"))
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, 27, stream.Height())
}

func Test_UIScrollKeysScrollLogViewWhilePromptClosed(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("k"))
	press(model, runtime, keys("j"))

	testutil.Equal(t, fmt.Sprint([]int{-1, -1, 1}), fmt.Sprint(stream.Scrolls()))
}

func Test_UIFollowKeyMakesLogViewFollowingWhilePromptClosed(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("k"))
	press(model, runtime, keys("G"))

	testutil.Equal(t, 1, stream.Follows())
}

func Test_UIScrollKeysAreTypedWhilePromptOpen(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("jk"))

	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "jk"))
}

func Test_UIHelpKeyOpensHelpInPlaceOfLogView(t *testing.T) {
	model, stream, runtime := startUI(t)
	stream.Notify(t, "a log line")
	model = runtime.Settle(model)

	model, _ = press(model, runtime, keys("h"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, true, strings.Contains(view, "Scroll the Log View or the Help down"))
	testutil.Equal(t, false, strings.Contains(view, "a log line"))
}

func Test_UIEscClosesHelpAndRestoresLogView(t *testing.T) {
	model, stream, runtime := startUI(t)
	stream.Notify(t, "a log line")
	model = runtime.Settle(model)

	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, esc)

	view := ansi.Strip(model.View())
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	testutil.Equal(t, false, strings.Contains(view, "Scroll the Log View or the Help down"))
	testutil.Equal(t, true, strings.Contains(view, "a log line"))
}

func Test_UIScrollKeysScrollHelpNotLogViewWhileHelpOpen(t *testing.T) {
	// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 Help rows
	model, stream, runtime := startSizedUI(t, 80, 5)

	model, _ = press(model, runtime, keys("h"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, keys("k"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, false, strings.Contains(view, "Open the Help"))
	testutil.Equal(t, true, strings.Contains(view, "Quit logctrl"))
	testutil.Equal(t, true, strings.Contains(view, "Open the Prompt"))
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
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("h"))

			model, quit := press(model, runtime, key)

			testutil.Equal(t, false, quit)
			testutil.Equal(t, 0, stream.Follows())
			testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "Scroll the Log View or the Help down"))
			// the Prompt stayed closed: 30 rows - 1 toolbar row - 2 border rows
			testutil.Equal(t, 27, stream.Height())
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
	model, stream, runtime := startRemappedUI(t, 80, 30, remapped)

	model, _ = press(model, runtime, tab)
	openedOnBuiltIn := stream.Height() != 27
	model, _ = press(model, runtime, keys("p"))
	openedOnRemapped := stream.Height() != 27
	model, _ = press(model, runtime, esc)
	closedOnBuiltIn := stream.Height() == 27
	press(model, runtime, tea.KeyMsg{Type: tea.KeyCtrlX})
	closedOnRemapped := stream.Height() == 27

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
	model, stream, runtime := startRemappedUI(t, 80, 30, remapped)

	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("j"))
	model, _ = press(model, runtime, keys("u"))
	model, _ = press(model, runtime, keys("u"))
	press(model, runtime, keys("n"))

	testutil.Equal(t, fmt.Sprint([]int{-1, -1, 1}), fmt.Sprint(stream.Scrolls()))
}

func Test_UIFollowsOnRemappedKeyNotTheBuiltInOne(t *testing.T) {
	model, stream, runtime := startRemappedUI(t, 80, 30, remapped)

	model, _ = press(model, runtime, keys("G"))
	followedOnBuiltIn := stream.Follows()
	press(model, runtime, tea.KeyMsg{Type: tea.KeyEnd})

	testutil.Equal(t, 0, followedOnBuiltIn)
	testutil.Equal(t, 1, stream.Follows())
}

func Test_UIScrollsHelpOnRemappedKeysNotTheBuiltInOnes(t *testing.T) {
	// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 Help rows
	model, stream, runtime := startRemappedUI(t, 80, 5, remapped)
	model, _ = press(model, runtime, keys("?"))

	model, _ = press(model, runtime, keys("j"))
	scrolledOnBuiltIn := !strings.Contains(ansi.Strip(model.View()), "Open the Help")
	model, _ = press(model, runtime, keys("n"))
	model, _ = press(model, runtime, keys("n"))
	model, _ = press(model, runtime, keys("u"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, false, scrolledOnBuiltIn)
	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, false, strings.Contains(view, "Open the Help"))
	testutil.Equal(t, true, strings.Contains(view, "Quit logctrl"))
	testutil.Equal(t, true, strings.Contains(view, "Open the Prompt"))
}

func Test_UISearchKeyOpensSearchBarAndShrinksLogView(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("/"))
	model, quit := press(model, runtime, keys("q.*jk"))

	testutil.Equal(t, false, quit)
	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 Search Bar rows - 2 border rows
	testutil.Equal(t, 25, stream.Height())
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: q.*jk"))
}

func Test_UIEnterMakesSearchBarReadOnly(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("err"))
	model, _ = press(model, runtime, enter)
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("x"))
	model, _ = press(model, runtime, keys("G"))

	testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
	testutil.Equal(t, 2, stream.Follows()) // on enter, then G
	testutil.Equal(t, 25, stream.Height())
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
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("/"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, ctrlL)
			closedView := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(closedView, "/: err"))
			testutil.Equal(t, 30, lipgloss.Height(model.View()))
			testutil.Equal(t, 27, stream.Height())
			testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))

			model, _ = press(model, runtime, keys("/"))
			testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "/: err"))
		})
	}
}

func Test_UISearchKeyMakesReadOnlySearchBarEditableKeepingRegex(t *testing.T) {
	model, stream, runtime := startUI(t)
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("err"))
	model, _ = press(model, runtime, enter)

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("ork"))

	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, 25, stream.Height())
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: errork"))
}

func Test_UIEscCancelsTypingInSearchBar(t *testing.T) {
	for _, test := range []struct {
		name     string
		typed    []tea.KeyMsg // after opening the Search Bar, before Esc
		height   int
		expected string // in the view after Esc, if anything
	}{
		{name: "first edit closes the Search Bar", typed: []tea.KeyMsg{keys("err")}, height: 27},
		{name: "re-edit restores the entered regex", typed: []tea.KeyMsg{keys("err"), enter, keys("/"), keys("or")}, height: 25, expected: "/: err"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("/"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, esc)
			view := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(view, "/: erro"))
			testutil.Equal(t, test.expected != "", strings.Contains(view, "/: err"))
			testutil.Equal(t, test.height, stream.Height())
			testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
		})
	}
}

func Test_UIEnterIsIgnoredOnEmptySearchBar(t *testing.T) {
	model, stream, runtime := startUI(t)
	model, _ = press(model, runtime, keys("/"))

	model, _ = press(model, runtime, enter)
	model, _ = press(model, runtime, keys("k"))

	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: k"))
}

func Test_UISearchStartsWhenRegexIsEntered(t *testing.T) {
	model, stream, runtime := startUI(t)
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("err"))

	testutil.Equal(t, "[]", updates(stream))

	model, _ = press(model, runtime, enter)
	testutil.Equal(t, "[SearchFor{err}]", updates(stream))

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("or"))
	_, _ = press(model, runtime, enter)
	testutil.Equal(t, "[SearchFor{err} SearchFor{error}]", updates(stream))
}

func Test_UICloseSearchKeyEndsSearch(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")

	_, _ = press(model, runtime, ctrlL)

	testutil.Equal(t, "[SearchFor{err} SearchFor{}]", updates(stream))
}

func Test_UICloseSearchKeyFollowsOnceRegexIsEntered(t *testing.T) {
	for _, test := range []struct {
		name  string
		regex string
		typed []tea.KeyMsg // after entering the regex, before closing the Search Bar
	}{
		{name: "while read-only", regex: "err"},
		{name: "while editing", regex: "err", typed: []tea.KeyMsg{keys("/"), keys("or")}},
		{name: "after a bad regex", regex: "err("},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model = enterSearch(model, runtime, test.regex)
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}
			model, _ = press(model, runtime, keys("k"))
			follows := stream.Follows()

			_, _ = press(model, runtime, ctrlL)

			testutil.Equal(t, follows+1, stream.Follows())
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
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("k"))
			model, _ = press(model, runtime, keys("/"))
			model, _ = press(model, runtime, keys("err"))

			_, _ = press(model, runtime, test.close)

			testutil.Equal(t, 0, stream.Follows())
		})
	}
}

func Test_UIEscKeepsEnteredSearch(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys("or"))

	_, _ = press(model, runtime, esc)

	testutil.Equal(t, "[SearchFor{err}]", updates(stream))
}

func Test_UIBadRegexShowsItsErrorInRedInSearchBar(t *testing.T) {
	model, stream, runtime := startUI(t)

	model = enterSearch(model, runtime, "err(")
	model, _ = press(model, runtime, keys("k"))

	testutil.Equal(t, "[SearchFor{err(}]", updates(stream))
	testutil.Equal(t, true, strings.Contains(model.View(),
		"\x1b[31merror parsing regexp: missing closing ): `err(`\x1b[0m"))
	testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
}

func Test_UISearchKeyAfterBadRegexBringsItBackAndEscShowsErrorAgain(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err(")

	model, _ = press(model, runtime, keys("/"))
	editView := ansi.Strip(model.View())
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, true, strings.Contains(editView, "/: err("))
	testutil.Equal(t, false, strings.Contains(editView, "error parsing regexp"))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: error parsing regexp: missing closing ): `err(`"))
	testutil.Equal(t, 25, stream.Height())
}

func Test_UIFixedRegexReplacesError(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err(")

	model, _ = press(model, runtime, keys("/"))
	model, _ = press(model, runtime, keys(")"))
	model, _ = press(model, runtime, enter)

	testutil.Equal(t, "[SearchFor{err(} SearchFor{err()}]", updates(stream))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err()"))
	testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "error parsing regexp"))
}

func Test_UILogViewRedrawsWhenSearchStartsOrEnds(t *testing.T) {
	for _, test := range []struct {
		name    string
		regex   string
		closing bool // close the Search Bar after entering `regex`
	}{
		{name: "Search starts", regex: "err"},
		{name: "bad regex ends the Search", regex: "err("},
		{name: "Search Bar closes", regex: "err", closing: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			if test.closing {
				model = enterSearch(model, runtime, test.regex)
			}

			stream.SetWindow("error 1") // as the Executor would show it after the change
			if test.closing {
				model, _ = press(model, runtime, ctrlL)
			} else {
				model = enterSearch(model, runtime, test.regex)
			}

			testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "error 1"))
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
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")

	model, _ = press(model, runtime, tab)
	model, _ = press(model, runtime, keys("cmd"))
	promptView := ansi.Strip(model.View())
	promptHeight := stream.Height()
	model, _ = press(model, runtime, esc)

	testutil.Equal(t, 30, lipgloss.Height(promptView))
	// 30 rows - 1 toolbar row - 2 Search Bar rows - 16 prompt rows - 2 border rows
	testutil.Equal(t, 9, promptHeight)
	testutil.Equal(t, true, strings.Index(promptView, "/: err") < strings.Index(promptView, "cmd"))
	testutil.Equal(t, 25, stream.Height())
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
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, tab)
	for _, key := range []string{"a", "/", "b"} {
		model, _ = press(model, runtime, keys(key))
	}

	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "a/b"))
	// the Search Bar stayed closed: 30 rows - 1 toolbar row - 16 prompt rows - 2 border rows
	testutil.Equal(t, 11, stream.Height())
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
			model, stream, runtime := startRemappedUI(t, 80, 30, remapped)

			model, _ = press(model, runtime, keys("/"))
			openedOnBuiltIn := stream.Height() != 27
			model, _ = press(model, runtime, keys("s"))
			openedOnRemapped := stream.Height() != 27
			model, _ = press(model, runtime, keys("err"))
			model, _ = press(model, runtime, test.builtIn)
			closedOnBuiltIn := stream.Height() == 27
			press(model, runtime, test.remapped)
			closedOnRemapped := stream.Height() == 27

			testutil.Equal(t, false, openedOnBuiltIn)
			testutil.Equal(t, true, openedOnRemapped)
			testutil.Equal(t, false, closedOnBuiltIn)
			testutil.Equal(t, true, closedOnRemapped)
		})
	}
}

func Test_UIFilterKeyOpensFilterBarAndShrinksLogView(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("f"))
	model, quit := press(model, runtime, keys("q.*jk/"))

	testutil.Equal(t, false, quit)
	testutil.Equal(t, 0, len(stream.Scrolls()))
	testutil.Equal(t, 30, lipgloss.Height(model.View()))
	// 30 rows - 1 toolbar row - 2 Filter Bar rows - 2 border rows
	testutil.Equal(t, 25, stream.Height())
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "f: q.*jk/"))
}

func Test_UIEnterMakesFilterBarReadOnlyWithoutSearching(t *testing.T) {
	model, stream, runtime := startUI(t)

	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys("err"))
	model, _ = press(model, runtime, enter)
	model, _ = press(model, runtime, keys("k"))
	model, _ = press(model, runtime, keys("x"))

	view := ansi.Strip(model.View())
	testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
	testutil.Equal(t, "[]", updates(stream))
	testutil.Equal(t, 0, stream.Follows())
	testutil.Equal(t, 25, stream.Height())
	testutil.Equal(t, true, strings.Contains(view, "f: err"))
	testutil.Equal(t, false, strings.Contains(view, "errk"))
	testutil.Equal(t, false, strings.Contains(view, "/: err"))
}

func Test_UICloseFilterKeyClosesFilterBarAndRestoresLogView(t *testing.T) {
	for _, test := range []struct {
		name  string
		typed []tea.KeyMsg // after opening the Filter Bar, before closing it
	}{
		{name: "while typing", typed: []tea.KeyMsg{keys("err")}},
		{name: "while read-only", typed: []tea.KeyMsg{keys("err"), enter}},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("f"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, ctrlF)
			closedView := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(closedView, "f: err"))
			testutil.Equal(t, 30, lipgloss.Height(model.View()))
			testutil.Equal(t, 27, stream.Height())
			testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
			testutil.Equal(t, "[]", updates(stream))
			testutil.Equal(t, 0, stream.Follows())

			model, _ = press(model, runtime, keys("f"))
			testutil.Equal(t, false, strings.Contains(ansi.Strip(model.View()), "f: err"))
		})
	}
}

func Test_UIEscCancelsTypingInFilterBar(t *testing.T) {
	for _, test := range []struct {
		name     string
		typed    []tea.KeyMsg // after opening the Filter Bar, before Esc
		height   int
		expected string // in the view after Esc, if anything
	}{
		{name: "first edit closes the Filter Bar", typed: []tea.KeyMsg{keys("err")}, height: 27},
		{name: "re-edit restores the entered regex", typed: []tea.KeyMsg{keys("err"), enter, keys("f"), keys("or")}, height: 25, expected: "f: err"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("f"))
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, esc)
			view := ansi.Strip(model.View())
			model, _ = press(model, runtime, keys("k"))

			testutil.Equal(t, false, strings.Contains(view, "f: erro"))
			testutil.Equal(t, test.expected != "", strings.Contains(view, "f: err"))
			testutil.Equal(t, test.height, stream.Height())
			testutil.Equal(t, fmt.Sprint([]int{-1}), fmt.Sprint(stream.Scrolls()))
		})
	}
}

func Test_UIFilterBarSitsAboveSearchBar(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, keys("f"))
	model, _ = press(model, runtime, keys("warn"))
	model, _ = press(model, runtime, enter)

	view := ansi.Strip(model.View())
	model, _ = press(model, runtime, keys("h"))
	helpView := ansi.Strip(model.View())

	testutil.Equal(t, 30, lipgloss.Height(view))
	// 30 rows - 1 toolbar row - 2 Filter Bar rows - 2 Search Bar rows - 2 border rows
	testutil.Equal(t, 23, stream.Height())
	testutil.Equal(t, fmt.Sprint([]string{topLine, "f: warn", topLine, "/: err"}), fmt.Sprint(lastRows(view, 4)))
	testutil.Equal(t, 30, lipgloss.Height(helpView))
	testutil.Equal(t, fmt.Sprint([]string{topLine, "f: warn", topLine, "/: err"}), fmt.Sprint(lastRows(helpView, 4)))
}

// topLine - the grey line along the top of a regex bar, as wide as the screen.
var topLine = strings.Repeat("─", 80)

func Test_UIRegexBarHasGreyTopLine(t *testing.T) {
	for _, open := range []string{"/", "f"} {
		t.Run(open, func(t *testing.T) {
			model, _, runtime := startUI(t)

			model, _ = press(model, runtime, keys(open))

			testutil.Equal(t, true, strings.Contains(model.View(), "\x1b[90m"+topLine+"\x1b[0m"))
		})
	}
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
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys(test.other[:1]))
			model, _ = press(model, runtime, keys(test.other[1:]))
			model, _ = press(model, runtime, enter)
			model, _ = press(model, runtime, test.open)

			model, _ = press(model, runtime, keys("a"))
			model, _ = press(model, runtime, test.key)
			model, _ = press(model, runtime, keys(test.other[:1]))
			model, _ = press(model, runtime, keys("b"))

			testutil.Equal(t, fmt.Sprint(test.expected), fmt.Sprint(lastRows(ansi.Strip(model.View()), 4)))
			testutil.Equal(t, 23, stream.Height())
		})
	}
}

func Test_UIRegexBarPromptIsBlackOnItsColourInEveryState(t *testing.T) {
	const (
		search = "\x1b[30;42m/:\x1b[0m "
		filter = "\x1b[30;45mf:\x1b[0m "
	)
	for _, test := range []struct {
		name   string
		typed  []tea.KeyMsg
		prompt string
	}{
		{name: "Search Bar while typing", typed: []tea.KeyMsg{keys("/"), keys("err")}, prompt: search},
		{name: "Search Bar while read-only", typed: []tea.KeyMsg{keys("/"), keys("err"), enter}, prompt: search},
		{name: "Search Bar showing an error", typed: []tea.KeyMsg{keys("/"), keys("err("), enter}, prompt: search},
		{name: "Filter Bar while typing", typed: []tea.KeyMsg{keys("f"), keys("warn")}, prompt: filter},
		{name: "Filter Bar while read-only", typed: []tea.KeyMsg{keys("f"), keys("warn"), enter}, prompt: filter},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, _, runtime := startUI(t)
			for _, key := range test.typed {
				model, _ = press(model, runtime, key)
			}

			testutil.Equal(t, true, strings.Contains(model.View(), test.prompt))
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
	up   = tea.KeyMsg{Type: tea.KeyUp}
	down = tea.KeyMsg{Type: tea.KeyDown}
)

// steps - the steps between Lines with a Match the UI asked for, in order.
func steps(stream *fakestream.Stream) string {
	var each []string
	for _, msg := range stream.Updates() {
		switch msg.(type) {
		case cmd.PreviousMatch:
			each = append(each, "previous")
		case cmd.NextMatch:
			each = append(each, "next")
		}
	}
	return fmt.Sprint(each)
}

func Test_UIStepKeysStepBetweenLinesWithMatchOnlyWhileRegexEnteredAndReadOnly(t *testing.T) {
	for _, test := range []struct {
		name     string
		before   []tea.KeyMsg // pressed after entering a Search, unless `noSearch`
		noSearch bool
		expected string
	}{
		{name: "regex entered", expected: "[previous next]"},
		{name: "no Search", noSearch: true, expected: "[]"},
		{name: "typing in the Search Bar", before: []tea.KeyMsg{keys("/")}, expected: "[]"},
		{name: "Help open", before: []tea.KeyMsg{keys("h")}, expected: "[]"},
		{name: "Prompt open", before: []tea.KeyMsg{tab}, expected: "[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			if !test.noSearch {
				model = enterSearch(model, runtime, "err")
			}
			for _, key := range test.before {
				model, _ = press(model, runtime, key)
			}

			model, _ = press(model, runtime, up)
			_, _ = press(model, runtime, down)

			testutil.Equal(t, test.expected, steps(stream))
		})
	}
}

func Test_UILogViewRedrawsAfterStep(t *testing.T) {
	for _, key := range []tea.KeyMsg{up, down} {
		t.Run(key.String(), func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model = enterSearch(model, runtime, "err")

			stream.SetWindow("err 5") // as the Executor would show it after the step
			model, _ = press(model, runtime, key)

			testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "err 5"))
		})
	}
}

// updates - every message the UI handed to the Passes, each with its type.
func updates(stream *fakestream.Stream) string {
	var each []string
	for _, msg := range stream.Updates() {
		each = append(each, strings.TrimPrefix(fmt.Sprintf("%T%v", msg, msg), "cmd."))
	}
	return fmt.Sprint(each)
}

func Test_UIEnterWithNewRegexMakesWindowFollowing(t *testing.T) {
	for _, test := range []struct {
		regex    string
		expected string
	}{
		{regex: "err", expected: "[SearchFor{err}]"},
		{regex: "err(", expected: "[SearchFor{err(}]"},
	} {
		t.Run(test.regex, func(t *testing.T) {
			model, stream, runtime := startUI(t)
			model, _ = press(model, runtime, keys("k"))
			model, _ = press(model, runtime, keys("/"))
			model, _ = press(model, runtime, keys(test.regex))

			stream.SetWindow("err 5") // as the Executor would show it
			model, _ = press(model, runtime, enter)

			testutil.Equal(t, test.expected, updates(stream))
			testutil.Equal(t, 1, stream.Follows())
			testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "err 5"))
		})
	}
}

func Test_UIEnterWithSameRegexChangesNothing(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, up) // Pins the Window on the Current Line

	model, _ = press(model, runtime, keys("/"))
	_, _ = press(model, runtime, enter)

	testutil.Equal(t, "[SearchFor{err} PreviousMatch{}]", updates(stream))
	testutil.Equal(t, 1, stream.Follows()) // only on the first enter
}

func Test_UIFollowKeyDropsCurrentLineKeepingSearch(t *testing.T) {
	model, stream, runtime := startUI(t)
	model = enterSearch(model, runtime, "err")
	model, _ = press(model, runtime, up)

	model, _ = press(model, runtime, keys("G"))

	testutil.Equal(t, "[SearchFor{err} PreviousMatch{} Follow{}]", updates(stream))
	testutil.Equal(t, 2, stream.Follows()) // on enter, then G
	testutil.Equal(t, true, strings.Contains(ansi.Strip(model.View()), "/: err"))
}
