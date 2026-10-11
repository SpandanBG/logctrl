---
status: accepted
---

# Keymap is embedded, and a user file is merged into it by Action

The built-in Keymap is `config/keymap.json`, embedded into the binary at build time so it can never be missing. If `~/.config/logCtrl/keymap.json` exists, it is merged into the built-in Keymap entry by entry, matched on `action`: a user entry replaces the key and, if given, the description of that Action, and Actions it leaves out keep their built-in entry. Users can therefore remap a key without rewriting descriptions, and new Actions added in later releases show up without the user editing their file. Each entry has the shape `{"key": "q", "action": "quit", "description": "Quit logctrl"}`, so a stable Action id, not the key, is what identifies an entry. Do not key entries by key.

In the user file only `action` is required; `key` and `description` fall back to the built-in. A user file that can't be read, isn't valid JSON, has an entry with a missing or unknown `action`, or leaves two Actions on the same key after merging stops logctrl before the UI opens, with an error naming the file. A broken config never fails silently.

For now the Keymap drives only what the Help and the Toolbar display. Key handling in the layout is still hard-coded, so a remapped key shows in the Help but does not work yet. Dispatching keys by Action from the merged Keymap is the planned next step.

## Considered Options

- Read the file only at runtime, from disk. Rejected: the binary would fail when the file is missing.
- A user file that replaces the whole list. Rejected: every user would have to restate every Action and description, and would miss Actions added later.
- Ignoring bad entries. Rejected: a typo like `scrol_up` would silently do nothing.
