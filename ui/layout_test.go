package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/internal/fakestream"
	"github.com/SpandanBG/logctrl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	tab = tea.KeyMsg{Type: tea.KeyTab}
	esc = tea.KeyMsg{Type: tea.KeyEsc}
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
	{"action": "follow", "key": "end"}
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

	model := tea.Model(newModel(stream, keymap))
	runtime.Run(model.Init())

	return runtime.Settle(model), stream, runtime
}

// press - sends `key` to the UI and settles it. Returns whether the UI quit.
func press(model tea.Model, runtime *testutil.Runtime, key tea.KeyMsg) (tea.Model, bool) {
	model, cmd := model.Update(key)
	quit := quits(cmd)
	runtime.Run(cmd)
	return runtime.Settle(model), quit
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
