---
status: accepted
---

# The Gutter Pass runs last, over context the other Passes leave on each Line

The Log View gets a Gutter, two cells wide down its left side, where marks about each Line are shown beside it, starting with `>` beside the Current Line. What a mark shows depends on what the Passes have worked out about the Line, and a mark must move with its Line on Scroll and as new Lines arrive. So each Line carries two new fields (in `reader`): `GutterContext`, the context the Passes leave for the Gutter, grouped by the Pass that sets it (for now `GutterContext.Search.Current`, set by the Search on the Current Line), and `Gutter`, a fixed array of `GutterWidth` (2) runes, 0 meaning an empty cell. A Gutter Pass, run by the Executor after the Search, turns each Line's `GutterContext` into its `Gutter` cells, so it sees each Line as every other Pass leaves it. The Search's port still reads Lines as the Passes before the Search leave them; the Gutter Pass never runs there. The Executor's `GetWindow` returns the shown text and the Gutter cells from the same read, each joined by newlines, so their rows always line up; the Log View draws the Gutter inside its border, on its own background, as tall as the Log View. The Search still remembers the Current Line's Line Number itself, as its stepping state, since a Line is copied fresh on every read. We rejected having the UI ask the Search for the Current Line and draw the mark itself (the UI would have to match Line Numbers to rows, and every later mark would need its own call), and letting each Pass write Gutter cells directly (two Passes could fight over a cell, and no one place would decide what the Gutter shows).

## Consequences

- The Search no longer runs last: the Gutter Pass runs after it. The Executor holds both apart from the other Passes (`NewExecutor(stream, search, gutter, others...)`), and hands every message to the Gutter last.
- A Pass that wants a mark in the Gutter adds its own context to `GutterContext`, and the Gutter Pass decides which cell it goes in. For now `>` goes in cell 1, beside the text.
- The Current Line is no longer shown bold; the `>` beside it marks it, and its Matches are highlighted like any other.
- `cmd.Stream.GetWindow` returns two strings, not one.
