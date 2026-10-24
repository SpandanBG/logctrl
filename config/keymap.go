// Package config holds the Keymap: the built-in one embedded at build time,
// merged with the user's own Keymap file if they have one.
package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// Action - something the user can make logctrl do with a key.
type Action string

const (
	OpenHelp      Action = "open_help"
	Quit          Action = "quit"
	OpenPrompt    Action = "open_prompt"
	Close         Action = "close"
	ScrollUp      Action = "scroll_up"
	ScrollDown    Action = "scroll_down"
	Follow        Action = "follow"
	OpenSearch    Action = "open_search"
	CloseSearch   Action = "close_search"
	PreviousMatch Action = "previous_match"
	NextMatch     Action = "next_match"
	OpenFilter    Action = "open_filter"
	CloseFilter   Action = "close_filter"
)

// Reserved - the keys that always Quit, outside the Keymap, so no Keymap file
// can leave the user without an exit.
var Reserved = []string{"ctrl+c", "ctrl+d"}

// whileTyping - the Actions honoured while the user types in the Prompt, the
// Search Bar or the Filter Bar, so they can't be on a key that would be typed.
var whileTyping = []Action{Close, CloseSearch, CloseFilter}

// Entry - one Action with the key that triggers it and its description.
type Entry struct {
	Key         string `json:"key"`
	Action      Action `json:"action"`
	Description string `json:"description"`
}

//go:embed keymap.json
var builtInKeymap []byte

// LoadKeymap - the built-in Keymap, merged with the user's Keymap file at
// `userPath` if it exists.
func LoadKeymap(userPath string) ([]Entry, error) {
	var keymap []Entry
	if err := json.Unmarshal(builtInKeymap, &keymap); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(userPath)
	if errors.Is(err, fs.ErrNotExist) {
		return keymap, nil
	}

	if err != nil {
		return nil, fmt.Errorf("keymap %s: %w", userPath, err)
	}

	var user []Entry
	if err := json.Unmarshal(content, &user); err != nil {
		return nil, fmt.Errorf("keymap %s: %w", userPath, err)
	}

	keymap, err = merge(keymap, user)
	if err != nil {
		return nil, fmt.Errorf("keymap %s: %w", userPath, err)
	}
	return keymap, nil
}

// ------------------------- Private

// merge - replaces the key and description of each Action in `keymap` with the
// ones `user` gives for it; ones `user` leaves empty are kept. Every Action in
// `user` must already be in `keymap`, and no two Actions may end up on one key
// or on a Reserved key, nor an Action honoured while typing on a printable key.
func merge(keymap, user []Entry) ([]Entry, error) {
	for _, override := range user {
		i := slices.IndexFunc(keymap, func(e Entry) bool {
			return e.Action == override.Action
		})
		if override.Action == "" {
			return nil, errors.New("entry without an action")
		}
		if i < 0 {
			return nil, fmt.Errorf("unknown action %q", override.Action)
		}
		if override.Key != "" {
			keymap[i].Key = override.Key
		}
		if override.Description != "" {
			keymap[i].Description = override.Description
		}
	}

	actionOf := map[string]Action{}
	for _, entry := range keymap {
		if !isKeyName(entry.Key) {
			return nil, fmt.Errorf("%s is on %q, which is not a key", entry.Action, entry.Key)
		}
		if slices.Contains(whileTyping, entry.Action) && isPrintable(entry.Key) {
			return nil, fmt.Errorf("%s is on %q, which would be typed in the Prompt, the Search Bar or the Filter Bar", entry.Action, entry.Key)
		}
		if slices.Contains(Reserved, entry.Key) {
			return nil, fmt.Errorf("%s is on reserved key %q", entry.Action, entry.Key)
		}
		if other, taken := actionOf[entry.Key]; taken {
			return nil, fmt.Errorf("%s and %s are both on key %q", other, entry.Action, entry.Key)
		}
		actionOf[entry.Key] = entry.Action
	}
	return keymap, nil
}

// isKeyName - whether `key` is a single character or a key name BubbleTea
// reports, either optionally prefixed with "alt+".
func isKeyName(key string) bool {
	key = strings.TrimPrefix(key, "alt+")
	if utf8.RuneCountInString(key) == 1 {
		return true
	}
	// key types are small negative (special keys) or control codes
	for keyType := tea.KeyType(-256); keyType < 128; keyType++ {
		if keyType != tea.KeyRunes && keyType.String() == key {
			return true
		}
	}
	return false
}

// isPrintable - whether `key` is a single printable character, space included.
func isPrintable(key string) bool {
	r, size := utf8.DecodeRuneInString(key)
	return size == len(key) && unicode.IsPrint(r)
}
