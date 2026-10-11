package components_test

import (
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var keymap = []config.Entry{
	{Key: "h", Action: config.OpenHelp, Description: "Open the Help"},
	{Key: "q", Action: config.Quit, Description: "Quit logctrl"},
	{Key: "k", Action: config.ScrollUp, Description: "Scroll the Log View up"},
}

// newSizedHelp - the Help in the area below a 1 row Toolbar of a `width` x
// `height` terminal.
func newSizedHelp(keymap []config.Entry, width, height int) tea.Model {
	help := components.NewHelp(ui.SizeRatio(1), ui.SizeModifier(-1), keymap)
	help, _ = help.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return help
}

// rows - the Help's visible lines, without styling or padding.
func rows(help tea.Model) []string {
	var rows []string
	for _, line := range strings.Split(ansi.Strip(help.View()), "\n") {
		if line = strings.Trim(line, " │─╭╮╰╯"); line != "" {
			rows = append(rows, strings.Join(strings.Fields(line), " "))
		}
	}
	return rows
}

func Test_HelpShowsEveryKeyWithItsDescription(t *testing.T) {
	help := newSizedHelp(keymap, 80, 30)

	testutil.Equal(t, 80, lipgloss.Width(help.View()))
	testutil.Equal(t, 29, lipgloss.Height(help.View()))
	testutil.Equal(t, strings.Join([]string{
		"h Open the Help",
		"q Quit logctrl",
		"k Scroll the Log View up",
	}, "\n"), strings.Join(rows(help), "\n"))
}

func Test_HelpScrollKeysScrollRowsWithinBoundsTableDriven(t *testing.T) {
	for _, test := range []struct {
		name     string
		keys     string
		expected []string
	}{
		{name: "starts at the top", keys: "", expected: []string{"h Open the Help", "q Quit logctrl"}},
		{name: "j scrolls down", keys: "j", expected: []string{"q Quit logctrl", "k Scroll the Log View up"}},
		{name: "j stops at the bottom", keys: "jj", expected: []string{"q Quit logctrl", "k Scroll the Log View up"}},
		{name: "k scrolls up", keys: "jk", expected: []string{"h Open the Help", "q Quit logctrl"}},
		{name: "k stops at the top", keys: "k", expected: []string{"h Open the Help", "q Quit logctrl"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 of 3 rows
			help := newSizedHelp(keymap, 80, 5)

			for _, key := range test.keys {
				help, _ = help.Update(keys(string(key)))
			}

			testutil.Equal(t, strings.Join(test.expected, "\n"), strings.Join(rows(help), "\n"))
		})
	}
}

func Test_HelpStartsAtTheTopWhenOpened(t *testing.T) {
	help := newSizedHelp(keymap, 80, 5)
	help, _ = help.Update(keys("j"))

	help, _ = help.Update(components.TeaHelpOpen{})

	testutil.Equal(t, "h Open the Help\nq Quit logctrl", strings.Join(rows(help), "\n"))
}
