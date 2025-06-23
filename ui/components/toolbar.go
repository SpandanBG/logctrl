package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	helpText = Grey_Color + "Quick Help:\t\t" +
		Magenta_Color + "q" + Black_Color + ":Quit" +
		Reset_Color
)

var (
	toolbarStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("43")).
		Foreground(lipgloss.Color("0"))
)

type toolbar struct {
	wRatio float32
	hRatio float32
	view   viewport.Model
	ready  bool
}

func NewToolbar(wRatio, hRatio float32) tea.Model {
	return toolbar{
		wRatio: wRatio,
		hRatio: hRatio,
	}
}

func (t toolbar) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
	)
}

func (t toolbar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return t.updateViewSize(msg)
	}

	return t, nil
}

func (t toolbar) View() string {
	return toolbarStyle.Render(t.view.View())
}

// ------------------------- Private
func (t toolbar) updateViewSize(size tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	w := int(t.wRatio*float32(size.Width)) - toolbarStyle.GetHorizontalFrameSize()
	h := int(t.hRatio*float32(size.Height)) - toolbarStyle.GetVerticalFrameSize()

	if t.ready {
		t.view.Width = w
		t.view.Height = h
	} else {
		t.view = viewport.New(w, h)
		t.ready = true

		t.view.SetContent(helpText)
	}

	return t, nil
}
