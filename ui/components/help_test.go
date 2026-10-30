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

// keymap - a Keymap of two entries for the Help to list.
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

func Test_HelpShowsEveryKeyWithItsDescriptionThenTheReservedQuitKeys(t *testing.T) {
	help := newSizedHelp(keymap, 80, 30)

	testutil.Equal(t, 80, lipgloss.Width(help.View()))
	testutil.Equal(t, 29, lipgloss.Height(help.View()))
	testutil.Equal(t, strings.Join([]string{
		"h Open the Help",
		"q Quit logctrl",
		"k Scroll the Log View up",
		"ctrl+c/ctrl+d Quit logctrl",
	}, "\n"), strings.Join(rows(help), "\n"))
}

func Test_HelpScrollMovesRowsWithinBoundsTableDriven(t *testing.T) {
	for _, test := range []struct {
		name     string
		scrolls  []components.TeaHelpScroll
		expected []string
	}{
		{name: "starts at the top", scrolls: nil, expected: []string{"h Open the Help", "q Quit logctrl"}},
		{name: "down", scrolls: []components.TeaHelpScroll{1}, expected: []string{"q Quit logctrl", "k Scroll the Log View up"}},
		{name: "down stops at the bottom", scrolls: []components.TeaHelpScroll{1, 1, 1}, expected: []string{"k Scroll the Log View up", "ctrl+c/ctrl+d Quit logctrl"}},
		{name: "up", scrolls: []components.TeaHelpScroll{1, -1}, expected: []string{"h Open the Help", "q Quit logctrl"}},
		{name: "up stops at the top", scrolls: []components.TeaHelpScroll{-1}, expected: []string{"h Open the Help", "q Quit logctrl"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// 5 rows - 1 toolbar row - 2 border rows leaves room for 2 of 4 rows
			help := newSizedHelp(keymap, 80, 5)

			for _, scroll := range test.scrolls {
				help, _ = help.Update(scroll)
			}

			testutil.Equal(t, strings.Join(test.expected, "\n"), strings.Join(rows(help), "\n"))
		})
	}
}

func Test_HelpStartsAtTheTopWhenOpened(t *testing.T) {
	help := newSizedHelp(keymap, 80, 5)
	help, _ = help.Update(components.TeaHelpScroll(1))

	help, _ = help.Update(components.TeaHelpOpen{})

	testutil.Equal(t, "h Open the Help\nq Quit logctrl", strings.Join(rows(help), "\n"))
}
