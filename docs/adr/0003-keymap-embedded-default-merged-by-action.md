---
status: accepted
---

# Keymap is embedded, and a user file is merged into it by Action

The built-in Keymap is `config/keymap.json`, embedded into the binary at build time so it can never be missing. If `~/.config/logCtrl/keymap.json` exists, it is merged into the built-in Keymap entry by entry, matched on `action`: a user entry replaces the key and, if given, the description of that Action, and Actions it leaves out keep their built-in entry. Users can therefore remap a key without rewriting descriptions, and new Actions added in later releases show up without the user editing their file. Each entry has the shape `{"key": "q", "action": "quit", "description": "Quit logctrl"}`, so a stable Action id, not the key, is what identifies an entry. Do not key entries by key.

In the user file only `action` is required; `key` and `description` fall back to the built-in. A user file that can't be read, isn't valid JSON, has an entry with a missing or unknown `action`, or leaves two Actions on the same key after merging stops logctrl before the UI opens, with an error naming the file. A broken config never fails silently, so these also stop logctrl:

- a key that is neither a single character nor a key name BubbleTea reports (`tab`, `esc`, `enter`, `up`, `ctrl+a`, ...), since a typo like `Tab` would never match a keypress;
- a key on `ctrl+c` or `ctrl+d`, which are reserved;
- `close` on a single printable character (space included), since `close` is the only Action honoured while the Prompt is open and every other key is typed into it. The same goes for `close_search`, which is honoured, along with `close`, while the user types in the Search Bar.

Each Action has one key. Keys are dispatched by Action from the merged Keymap: the layout turns a keypress into an Action once, and components receive Action-level messages, never raw keys. `ctrl+c` and `ctrl+d` are outside the Keymap and always Quit, from the Log View or the Help (not the Prompt), so no user file can leave the user without an exit; the Help lists them in a fixed last row.

## Considered Options

- Read the file only at runtime, from disk. Rejected: the binary would fail when the file is missing.
- A user file that replaces the whole list. Rejected: every user would have to restate every Action and description, and would miss Actions added later.
- Ignoring bad entries. Rejected: a typo like `scrol_up` would silently do nothing.
- A list of keys per Action (e.g. `q` and `ctrl+d` for Quit). Rejected for now: one key keeps the schema, the Help rows and the duplicate-key check simple; a list can be added later without breaking existing files.
- Making `ctrl+c`/`ctrl+d` remappable. Rejected: a bad user file could leave no way to quit.
