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
)

// Action - something the user can make logctrl do with a key.
type Action string

const (
	OpenHelp   Action = "open_help"
	Quit       Action = "quit"
	OpenPrompt Action = "open_prompt"
	Close      Action = "close"
	ScrollUp   Action = "scroll_up"
	ScrollDown Action = "scroll_down"
)

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
// `user` must already be in `keymap`, and no two Actions may end up on one key.
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
		if other, taken := actionOf[entry.Key]; taken {
			return nil, fmt.Errorf("%s and %s are both on key %q", other, entry.Action, entry.Key)
		}
		actionOf[entry.Key] = entry.Action
	}
	return keymap, nil
}
