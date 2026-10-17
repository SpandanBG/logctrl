package components

import (
	"slices"
	"strings"

	"github.com/SpandanBG/logctrl/config"
	ui "github.com/SpandanBG/logctrl/ui/utils"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ----- Public tea.Msg
type TeaHelpOpen struct{}

// TeaHelpSizeUpdate - gives the Help a new height modifier.
type TeaHelpSizeUpdate struct {
	Height ui.SizeI
}

var (
	helpStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("43")).
			Padding(0, 1)
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true)
)

type help struct {
	width  ui.SizeI          // width modifier
	height ui.SizeI          // height modifier
	area   tea.WindowSizeMsg // the area the box is centered in
	rows   string            // one row per Keymap entry
	view   viewport.Model    // the visible rows
}

func NewHelp(width, height ui.SizeI, keymap []config.Entry) tea.Model {
	return help{
		width:  width,
		height: height,
		rows:   renderRows(withReservedQuit(keymap)),
	}
}

func (h help) Init() tea.Cmd {
	return tea.WindowSize()
}

func (h help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h.updateSize(msg)
	case TeaHelpOpen:
		h.view.GotoTop()
		return h, nil
	case TeaScroll:
		return h.scroll(int(msg))
	case TeaHelpSizeUpdate:
		h.height = msg.Height
		return h, tea.WindowSize()
	}
	return h, nil
}

func (h help) View() string {
	return lipgloss.Place(
		h.area.Width, h.area.Height,
		lipgloss.Center, lipgloss.Center,
		helpStyle.Render(h.view.View()),
	)
}

// ------------------------- Private

// scroll - moves the rows `n` up (negative) or down, stopping at the
// first/last row.
func (h help) scroll(n int) (tea.Model, tea.Cmd) {
	if n < 0 {
		h.view.ScrollUp(-n)
	} else {
		h.view.ScrollDown(n)
	}
	return h, nil
}

// withReservedQuit - `keymap` with a last entry for the Reserved keys, which
// always Quit.
func withReservedQuit(keymap []config.Entry) []config.Entry {
	return append(slices.Clone(keymap), config.Entry{
		Key:         strings.Join(config.Reserved, "/"),
		Action:      config.Quit,
		Description: "Quit logctrl",
	})
}

// renderRows - one row per entry: the key, padded to the widest key, then its
// description.
func renderRows(keymap []config.Entry) string {
	keyWidth := 0
	for _, entry := range keymap {
		keyWidth = max(keyWidth, lipgloss.Width(entry.Key))
	}

	rows := make([]string, len(keymap))
	for i, entry := range keymap {
		key := helpKeyStyle.Width(keyWidth).Render(entry.Key)
		rows[i] = key + "  " + entry.Description
	}
	return strings.Join(rows, "\n")
}

// updateSize - fits the box to its rows, but no taller or wider than the area.
func (h help) updateSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	h.area = ui.ModifySize(size, h.width, h.height)

	w := min(lipgloss.Width(h.rows), h.area.Width-helpStyle.GetHorizontalFrameSize())
	ht := min(lipgloss.Height(h.rows), h.area.Height-helpStyle.GetVerticalFrameSize())

	h.view = viewport.New(w, ht)
	h.view.SetContent(h.rows)
	return h, nil
}
