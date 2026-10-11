package components_test

import (
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/ui/components"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func Test_ToolbarShowsKeyHelpAtItsSize(t *testing.T) {
	toolbar := components.NewToolbar(ui.SizeRatio(1), ui.SizeFixed(1))

	toolbar, _ = toolbar.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	view := toolbar.View()

	testutil.Equal(t, 80, lipgloss.Width(view))
	testutil.Equal(t, 1, lipgloss.Height(view))
	testutil.Equal(t, true, strings.Contains(ansi.Strip(view), "q:Quit"))
}
