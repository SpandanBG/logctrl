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

func Test_ToolbarShowsOnlyHelpScrollSearchFilterAndMatchKeysFromKeymapAtItsSize(t *testing.T) {
	toolbar := components.NewToolbar(ui.SizeRatio(1), ui.SizeFixed(1), []config.Entry{
		{Key: "q", Action: config.Quit, Description: "Quit logctrl"},
		{Key: "?", Action: config.OpenHelp, Description: "Open the Help"},
		{Key: "p", Action: config.ScrollUp, Description: "Scroll the Log View up"},
		{Key: "n", Action: config.ScrollDown, Description: "Scroll the Log View down"},
		{Key: "G", Action: config.Follow, Description: "Follow the newest lines"},
		{Key: "f", Action: config.OpenSearch, Description: "Open the Search Bar"},
		{Key: "ctrl+o", Action: config.CloseSearch, Description: "Close the Search Bar"},
		{Key: "N", Action: config.PreviousMatch, Description: "Step to the previous Line with a Match"},
		{Key: "m", Action: config.NextMatch, Description: "Step to the next Line with a Match"},
		{Key: "x", Action: config.OpenFilter, Description: "Open the Filter Bar"},
		{Key: "ctrl+x", Action: config.CloseFilter, Description: "Close the Filter Bar"},
	})

	toolbar, _ = toolbar.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	view := toolbar.View()

	testutil.Equal(t, 80, lipgloss.Width(view))
	testutil.Equal(t, 1, lipgloss.Height(view))
	testutil.Equal(t, "Quick Help: ?:Help n/p:Scroll f:Search x:Filter m/N:Match", strings.Join(strings.Fields(ansi.Strip(view)), " "))
}
