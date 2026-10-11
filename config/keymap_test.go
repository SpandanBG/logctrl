package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SpandanBG/logctrl/config"
	"github.com/SpandanBG/logctrl/internal/testutil"
)

var builtIn = []config.Entry{
	{Key: "h", Action: config.OpenHelp, Description: "Open the Help"},
	{Key: "q", Action: config.Quit, Description: "Quit logctrl"},
	{Key: "tab", Action: config.OpenPrompt, Description: "Open the Prompt"},
	{Key: "esc", Action: config.Close, Description: "Close the Prompt or the Help"},
	{Key: "k", Action: config.ScrollUp, Description: "Scroll the Log View up"},
	{Key: "j", Action: config.ScrollDown, Description: "Scroll the Log View down"},
}

// userFile - writes `content` as the user's Keymap file and returns its path.
func userFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "keymap.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func Test_KeymapIsBuiltInWithoutUserFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "keymap.json")

	keymap, err := config.LoadKeymap(missing)

	testutil.Equal(t, nil, err)
	testutil.Equal(t, fmt.Sprint(builtIn), fmt.Sprint(keymap))
}

func Test_KeymapMergesUserFileByAction(t *testing.T) {
	path := userFile(t, `[
		{"action": "quit", "key": "x"},
		{"action": "scroll_up", "description": "Older lines"}
	]`)

	keymap, err := config.LoadKeymap(path)

	expected := append([]config.Entry(nil), builtIn...)
	expected[1].Key = "x"
	expected[4].Description = "Older lines"
	testutil.Equal(t, nil, err)
	testutil.Equal(t, fmt.Sprint(expected), fmt.Sprint(keymap))
}

func Test_KeymapRejectsBadUserFileTableDriven(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
	}{
		{name: "invalid json", content: `[{"action": "quit",`},
		{name: "missing action", content: `[{"key": "x"}]`},
		{name: "unknown action", content: `[{"action": "scrol_up", "key": "x"}]`},
		{name: "two actions on one key", content: `[{"action": "scroll_up", "key": "q"}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := userFile(t, test.content)

			_, err := config.LoadKeymap(path)

			testutil.Equal(t, true, err != nil && strings.Contains(err.Error(), path))
		})
	}
}

func Test_KeymapRejectsUnreadableUserFile(t *testing.T) {
	dir := t.TempDir() // a directory can't be read as a file

	_, err := config.LoadKeymap(dir)

	testutil.Equal(t, true, err != nil && strings.Contains(err.Error(), dir))
}
