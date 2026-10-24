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
	{Key: "esc", Action: config.Close, Description: "Close the Prompt or the Help, or cancel typing a Search"},
	{Key: "k", Action: config.ScrollUp, Description: "Scroll the Log View or the Help up"},
	{Key: "j", Action: config.ScrollDown, Description: "Scroll the Log View or the Help down"},
	{Key: "G", Action: config.Follow, Description: "Follow the newest lines, dropping the Current Line"},
	{Key: "/", Action: config.OpenSearch, Description: "Open the Search Bar"},
	{Key: "ctrl+l", Action: config.CloseSearch, Description: "Close the Search Bar"},
	{Key: "up", Action: config.PreviousMatch, Description: "Step to the previous Line with a Match"},
	{Key: "down", Action: config.NextMatch, Description: "Step to the next Line with a Match"},
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
		{name: "action on reserved ctrl+c", content: `[{"action": "quit", "key": "ctrl+c"}]`},
		{name: "action on reserved ctrl+d", content: `[{"action": "open_help", "key": "ctrl+d"}]`},
		{name: "capitalised key name", content: `[{"action": "open_prompt", "key": "Tab"}]`},
		{name: "misspelt key name", content: `[{"action": "quit", "key": "ctrl-x"}]`},
		{name: "several characters", content: `[{"action": "quit", "key": "qq"}]`},
		{name: "close on a printable character", content: `[{"action": "close", "key": "x"}]`},
		{name: "close on space", content: `[{"action": "close", "key": " "}]`},
		{name: "close_search on a printable character", content: `[{"action": "close_search", "key": "x"}]`},
		{name: "close_search on space", content: `[{"action": "close_search", "key": " "}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := userFile(t, test.content)

			_, err := config.LoadKeymap(path)

			testutil.Equal(t, true, err != nil && strings.Contains(err.Error(), path))
		})
	}
}

func Test_KeymapAcceptsKeysBubbleTeaReportsTableDriven(t *testing.T) {
	for _, key := range []string{"x", "?", "enter", "left", "pgdown", "ctrl+x", "alt+x", "f5"} {
		t.Run(key, func(t *testing.T) {
			path := userFile(t, `[{"action": "quit", "key": "`+key+`"}]`)

			keymap, err := config.LoadKeymap(path)

			testutil.Equal(t, nil, err)
			testutil.Equal(t, key, keymap[1].Key)
		})
	}
}

func Test_KeymapAcceptsCloseOnKeyThatIsNotTyped(t *testing.T) {
	path := userFile(t, `[{"action": "close", "key": "ctrl+x"}]`)

	keymap, err := config.LoadKeymap(path)

	testutil.Equal(t, nil, err)
	testutil.Equal(t, "ctrl+x", keymap[3].Key)
}

func Test_KeymapRemapsFollow(t *testing.T) {
	path := userFile(t, `[{"action": "follow", "key": "end"}]`)

	keymap, err := config.LoadKeymap(path)

	testutil.Equal(t, nil, err)
	testutil.Equal(t, "end", keymap[6].Key)
}

func Test_KeymapRemapsStepsBetweenLinesWithMatch(t *testing.T) {
	path := userFile(t, `[{"action": "previous_match", "key": "N"}, {"action": "next_match", "key": "n"}]`)

	keymap, err := config.LoadKeymap(path)

	testutil.Equal(t, nil, err)
	testutil.Equal(t, "N n", keymap[9].Key+" "+keymap[10].Key)
}

func Test_KeymapRejectsUnreadableUserFile(t *testing.T) {
	dir := t.TempDir() // a directory can't be read as a file

	_, err := config.LoadKeymap(dir)

	testutil.Equal(t, true, err != nil && strings.Contains(err.Error(), dir))
}
