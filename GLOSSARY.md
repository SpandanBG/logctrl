# logctrl

A terminal log viewer: it consumes a stream of log lines from another process and lets the user watch them live or scroll back through recent history.

## Ingest

**Log Feed**:
The incoming stream of log lines produced by the process being watched.
_Avoid_: input, pipe, source

**Log File**:
The complete on-disk copy of everything received on the Log Feed during a session.
_Avoid_: temp file, log dump

**Line**:
One entry of the Log Feed: its text, and what logctrl works out about it, such as its formatting or where a Search matches it.
_Avoid_: log line, entry, record, row

**Line Number**:
A line's position in the Log Feed since the session started; it never changes, even after the line leaves the Live Buffer.
_Avoid_: index, offset, position

**Total**:
The number of lines received on the Log Feed so far.
_Avoid_: count, length

**History**:
Every Line received on the Log Feed, from Line Number 0 to Total, whether held in the Live Buffer or read back from the Log File in Chunks.
_Avoid_: backlog, scrollback, buffer, log

**Line Index**:
The record of where in the Log File certain Line Numbers begin, kept up to date as lines arrive, so any line can be found without reading the Log File from the start.
_Avoid_: offset table, line map

## Display

**Live Buffer**:
The most recent lines held in memory, up to its Capacity.
_Avoid_: LAB, live access buffer, scrollback, history

**Chunk**:
A contiguous run of Line Numbers older than Oldest Held, read from the Log File into memory so the Window can show them.
_Avoid_: page, history buffer, archive

**Capacity**:
The maximum number of lines the Live Buffer holds before the oldest are evicted.
_Avoid_: buffer size, scrollback size

**Oldest Held**:
The Line Number of the oldest line still in the Live Buffer.

**Window**:
The contiguous range of Line Numbers currently shown to the user, as tall as the log view.
_Avoid_: RAB, random access buffer, viewport, page

**Following**:
The Window mode in which it always shows the newest lines; the user can return to it at any time with a single key, which also drops any Current Line. Entering a new Search regex, or ending the Search by closing the Search Bar, returns the Window to it too.
_Avoid_: live mode, tailing, auto-scroll

**Pinned**:
The Window mode in which it keeps showing the same Line Numbers while new lines arrive, even after they leave the Live Buffer.
_Avoid_: frozen, paused, locked

**Scroll**:
Moving the Window a number of lines up (toward older lines, as far back as the first line of the session) or down (toward newer lines); scrolling down to the newest line makes it Following again.
_Avoid_: move, step, navigate

**Search**:
Looking for lines that match a regex the user types, from the newest line toward older ones.
_Avoid_: find, grep, reverse search

**Match**:
A section of a Line's text that the Search regex matches; a Line may have several, or none.
_Avoid_: matched span, hit, result

**Current Line**:
The Line with a Match that the user has stepped to during a Search, marked out from the other Lines and kept in the middle of the Window; the user steps from it to the previous (older) or next (newer) Line with a Match. There is at most one: none until the user first steps during a Search, none while no Line matches, and none again once the user returns the Window to Following with its key.
_Avoid_: result, hit, cursor, selection, Current Match

**Filter**:
Showing only the Lines that match a regex the user types; the other Lines are hidden from the Log View.
_Avoid_: grep, search, hide rule

**Pass**:
Processing run over each Line in the Window before it is shown, such as formatting it or a Search; every Pass runs again whenever the Window changes. The Search runs over each Line as the formatting Passes leave it, and the marks in the Gutter are worked out last, from each Line as every other Pass leaves it.
_Avoid_: command, stage, step

**Log View**:
The on-screen panel that shows the Window.
_Avoid_: viewport, log panel

**Gutter**:
The shaded strip two columns wide down the left of the Log View, as tall as it and always shown, where marks about each Line are shown beside it, such as the one beside the Current Line; a mark moves with its Line.
_Avoid_: sign column, margin, ruler

**Toolbar**:
The bar at the top of the screen listing the available keys.
_Avoid_: help bar, status bar

**Prompt**:
The panel, opened and closed by the user, where they type a command to apply to the Log Feed.
_Avoid_: input, textarea, command line

**Help**:
The panel, opened and closed by the user, listing every key in the Keymap and what it does.
_Avoid_: help menu, help window, popup

**Search Bar**:
The one-line panel under the Log View where the user types the regex for a Search. Once entered, the regex stays on show, read-only, until the user closes the Search Bar; a regex that cannot be searched for shows why in its place, ready to be fixed.
_Avoid_: search box, input box, Prompt

**Filter Bar**:
The one-line panel under the Log View, above the Search Bar when both are shown, where the user types the regex for a Filter. Once entered, the regex stays on show, read-only, until the user closes the Filter Bar, which ends the Filter.
_Avoid_: filter box, input box, Search Bar

## Controls

**Action**:
Something the user can make logctrl do with a key, such as Quit, open the Help or Scroll down.
_Avoid_: command, binding

**Keymap**:
The list pairing each Action with the key that triggers it and a description of it; the user's own Keymap file may change the key or description of any Action.
_Avoid_: keybinds, shortcuts, key config
