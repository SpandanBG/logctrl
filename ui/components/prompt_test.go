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

// typeText - sends `text` to the model as typed keys.
func typeText(model tea.Model, text string) tea.Model {
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return model
}

func newSizedPrompt() tea.Model {
	prompt := components.NewPrompt(ui.SizeRatio(1), ui.SizeFixed(16))
	prompt, _ = prompt.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	return prompt
}

func Test_PromptTakesItsSize(t *testing.T) {
	view := newSizedPrompt().View()

	testutil.Equal(t, 80, lipgloss.Width(view))
	testutil.Equal(t, 16, lipgloss.Height(view))
}

func Test_PromptIgnoresTypingUntilFocused(t *testing.T) {
	prompt := typeText(newSizedPrompt(), "xyz")

	testutil.Equal(t, false, strings.Contains(ansi.Strip(prompt.View()), "xyz"))
}

func Test_PromptAcceptsTypingWhileFocused(t *testing.T) {
	prompt, _ := newSizedPrompt().Update(components.TeaPromptToggle{BringFocus: true})
	prompt = typeText(prompt, "xyz")

	testutil.Equal(t, true, strings.Contains(ansi.Strip(prompt.View()), "xyz"))
}

func Test_PromptKeepsTextButIgnoresTypingOnceUnfocused(t *testing.T) {
	prompt, _ := newSizedPrompt().Update(components.TeaPromptToggle{BringFocus: true})
	prompt = typeText(prompt, "xyz")
	prompt, _ = prompt.Update(components.TeaPromptToggle{BringFocus: false})
	prompt = typeText(prompt, "abc")

	view := ansi.Strip(prompt.View())
	testutil.Equal(t, true, strings.Contains(view, "xyz"))
	testutil.Equal(t, false, strings.Contains(view, "abc"))
}
