package components_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

var (
	edit  = components.TeaRegexEdit{}
	enter = tea.KeyMsg{Type: tea.KeyEnter}
)

var (
	// typed - opens a regex bar and types "err" in it.
	typed = []tea.Msg{edit, keys("err")}
	// entered - opens a regex bar and enters "err" in it.
	entered = []tea.Msg{edit, keys("err"), enter}
)

// newSizedBar - the regex bar `bar`, hidden, on an 80x30 terminal.
func newSizedBar(bar components.Bar) components.RegexBar {
	regexBar, _ := components.NewRegexBar(ui.SizeRatio(1), bar, testutil.Colors()).
		UpdateBar(tea.WindowSizeMsg{Width: 80, Height: 30})
	return regexBar
}

// send - `bar` after each of `msgs`, in order.
func send(bar components.RegexBar, msgs ...tea.Msg) components.RegexBar {
	for _, msg := range msgs {
		bar, _ = bar.UpdateBar(msg)
	}
	return bar
}

// promptRow - the row of `view` holding the prompt and regex, unstyled and
// without its padding, or "" if `view` is empty.
func promptRow(view string) string {
	rows := strings.Split(ansi.Strip(view), "\n")
	return strings.TrimSpace(rows[len(rows)-1])
}

// viewRows - how many rows `view` takes, none if it is empty.
func viewRows(view string) int {
	if view == "" {
		return 0
	}
	return strings.Count(view, "\n") + 1
}

// barState - what the layout reads from a regex bar, and its prompt row.
type barState struct {
	shown   bool
	typing  bool
	entered bool
	height  int
	row     string // "" while hidden
}

func Test_RegexBarIsHiddenTypingOrReadOnlyAfterEachMessage(t *testing.T) {
	for _, test := range []struct {
		name     string
		bar      components.Bar // the Search Bar unless set
		msgs     []tea.Msg
		expected barState
	}{
		{name: "hidden once built"},
		{name: "edit shows it typing", msgs: typed, expected: barState{shown: true, typing: true, height: 2, row: "/: err"}},
		{name: "enter makes it read-only", msgs: entered, expected: barState{shown: true, entered: true, height: 2, row: "/: err"}},
		{name: "enter makes the Filter Bar read-only", bar: components.FilterBar, msgs: entered, expected: barState{shown: true, entered: true, height: 2, row: "f: err"}},
		{name: "enter is ignored while empty", msgs: []tea.Msg{edit, enter, keys("k")}, expected: barState{shown: true, typing: true, height: 2, row: "/: k"}},
		{name: "edit keeps the entered regex", msgs: slices.Concat(entered, []tea.Msg{edit, keys("or")}), expected: barState{shown: true, typing: true, entered: true, height: 2, row: "/: error"}},
		{name: "cancel goes back to the entered regex", msgs: slices.Concat(entered, []tea.Msg{edit, keys("or"), components.TeaRegexCancel{}}), expected: barState{shown: true, entered: true, height: 2, row: "/: err"}},
		{name: "cancel with no regex entered hides it", msgs: slices.Concat(typed, []tea.Msg{components.TeaRegexCancel{}})},
		{name: "close while typing hides it", msgs: slices.Concat(typed, []tea.Msg{components.TeaRegexClose{}})},
		{name: "close while read-only hides it", msgs: slices.Concat(entered, []tea.Msg{components.TeaRegexClose{}})},
		{name: "edit after close starts empty", msgs: slices.Concat(entered, []tea.Msg{components.TeaRegexClose{}, edit}), expected: barState{shown: true, typing: true, height: 2, row: "/:"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			bar := send(newSizedBar(test.bar), test.msgs...)

			actual := barState{
				shown:   bar.Shown(),
				typing:  bar.Typing(),
				entered: bar.Entered(),
				height:  bar.Height(),
				row:     promptRow(bar.View()),
			}
			testutil.Equal(t, fmt.Sprintf("%+v", test.expected), fmt.Sprintf("%+v", actual))
			testutil.Equal(t, bar.Height(), viewRows(bar.View()))
		})
	}
}

func Test_RegexBarEnterSaysWhichBarTheRegexWasEnteredIn(t *testing.T) {
	for _, test := range []struct {
		name     string
		bar      components.Bar
		typed    string
		expected tea.Msg // from the command enter returns, nil if none
	}{
		{name: "Search Bar", bar: components.SearchBar, typed: "err", expected: components.TeaRegexEntered{Bar: components.SearchBar, Regex: "err"}},
		{name: "Filter Bar", bar: components.FilterBar, typed: "warn", expected: components.TeaRegexEntered{Bar: components.FilterBar, Regex: "warn"}},
		{name: "empty Search Bar sends nothing", bar: components.SearchBar},
	} {
		t.Run(test.name, func(t *testing.T) {
			bar := send(newSizedBar(test.bar), edit, keys(test.typed))

			_, cmd := bar.UpdateBar(enter)

			var msg tea.Msg
			if cmd != nil {
				msg = cmd()
			}
			testutil.Equal(t, test.expected, msg)
		})
	}
}

func Test_RegexBarShowsWhyTheRegexFailedInRedWhileReadOnly(t *testing.T) {
	const red = "\x1b[31m"
	failed := components.TeaRegexFailed{Err: errors.New("missing closing )")}
	long := components.TeaRegexFailed{Err: errors.New("missing closing ): `(" + strings.Repeat("x", 120) + "`")}
	for _, test := range []struct {
		name string
		msgs []tea.Msg // after entering "err("
		row  string
		red  bool // whether the row shows the error in red
	}{
		{name: "read-only", msgs: []tea.Msg{failed}, row: "/: missing closing )", red: true},
		{name: "editing brings the regex back", msgs: []tea.Msg{failed, edit}, row: "/: err("},
		{name: "cancel shows the error again", msgs: []tea.Msg{failed, edit, keys(")"), components.TeaRegexCancel{}}, row: "/: missing closing )", red: true},
		{name: "entering a new regex drops the error", msgs: []tea.Msg{failed, edit, keys(")"), enter}, row: "/: err()"},
		// 80 columns: "/: ", then 76 of the error and the "…".
		{name: "cut to the bar's width", msgs: []tea.Msg{long}, row: "/: missing closing ): `(" + strings.Repeat("x", 55) + "…", red: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			bar := send(newSizedBar(components.SearchBar), edit, keys("err("), enter)

			bar = send(bar, test.msgs...)

			view := bar.View()
			testutil.Equal(t, test.row, promptRow(view))
			testutil.Equal(t, test.red, strings.Contains(view, red+strings.TrimPrefix(test.row, "/: ")+"\x1b[0m"))
			testutil.Equal(t, bar.Height(), viewRows(view))
			if t.Failed() {
				t.Logf("view: %q", view)
			}
		})
	}
}

func Test_RegexBarPromptAndTopLineKeepTheirColorsAndRegexIsFaintOnlyWhileReadOnly(t *testing.T) {
	const (
		search = "\x1b[30;42m/:\x1b[0m "
		filter = "\x1b[30;45mf:\x1b[0m "
		faint  = "\x1b[2merr\x1b[0m"
	)
	topLine := "\x1b[90m" + strings.Repeat("─", 80) + "\x1b[0m" // grey, as wide as the bar
	for _, test := range []struct {
		name   string
		bar    components.Bar
		msgs   []tea.Msg
		prompt string
		faint  bool // whether the regex is dimmed
	}{
		{name: "Search Bar while typing", bar: components.SearchBar, msgs: typed, prompt: search},
		{name: "Search Bar while read-only", bar: components.SearchBar, msgs: entered, prompt: search, faint: true},
		{name: "Search Bar while editing again", bar: components.SearchBar, msgs: slices.Concat(entered, []tea.Msg{edit}), prompt: search},
		{name: "Search Bar showing an error", bar: components.SearchBar, msgs: slices.Concat(entered, []tea.Msg{components.TeaRegexFailed{Err: errors.New("bad")}}), prompt: search},
		{name: "Filter Bar while typing", bar: components.FilterBar, msgs: typed, prompt: filter},
		{name: "Filter Bar while read-only", bar: components.FilterBar, msgs: entered, prompt: filter, faint: true},
		{name: "Filter Bar while editing again", bar: components.FilterBar, msgs: slices.Concat(entered, []tea.Msg{edit}), prompt: filter},
	} {
		t.Run(test.name, func(t *testing.T) {
			view := send(newSizedBar(test.bar), test.msgs...).View()

			testutil.Equal(t, true, strings.Contains(view, test.prompt))
			testutil.Equal(t, true, strings.Contains(view, topLine))
			testutil.Equal(t, test.faint, strings.Contains(view, faint))
			if t.Failed() {
				t.Logf("view: %q", view)
			}
		})
	}
}
