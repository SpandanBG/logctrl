# logctrl

Watch, scroll, filter and run commands on live logs, right in your terminal.

## What it does

Pipe any process into `logctrl` and its log lines appear in the Log View as they arrive. Scroll up to read older lines: the Log View stays where you put it while new lines keep arriving. Scroll back down to the newest line and it follows the live logs again. Search the logs with a regex: every Match is highlighted, and you can step between the Lines that have one.

- Everything received is also kept in a Log File, `$TMPDIR/logCtrl_logFile.txt`, so nothing is lost when it scrolls out of reach.
- Quitting `logctrl` also stops the process piped into it.

## Install

With Go 1.23 or later:

```bash
go install github.com/SpandanBG/logctrl@latest
```

Or from source:

```bash
git clone https://github.com/SpandanBG/logctrl.git
cd logctrl
go build -o logctrl .
```

`logctrl` runs on macOS and Linux (it needs `/dev/tty`). Windows is not supported.

## Usage

```bash
tail -f app.log | logctrl
npm run dev 2>&1 | logctrl
```

Only standard output is piped by default. Add `2>&1` to see the process's errors too.

## Keys

Press `h` at any time to open the Help, which lists every key.

| Default key | Action id | What it does |
|---|---|---|
| `h` | `open_help` | Open the Help |
| `q` | `quit` | Quit logctrl |
| `tab` | `open_prompt` | Open the Prompt |
| `esc` | `close` | Close the Prompt or the Help, or cancel typing a Search |
| `k` | `scroll_up` | Scroll the Log View or the Help up (older lines) |
| `j` | `scroll_down` | Scroll the Log View or the Help down (newer lines) |
| `G` | `follow` | Follow the newest lines, dropping the Current Line |
| `/` | `open_search` | Open the Search Bar, or edit the regex in it |
| `ctrl+l` | `close_search` | Close the Search Bar, ending the Search and following the newest lines |
| `up` | `previous_match` | Step to the previous (older) Line with a Match |
| `down` | `next_match` | Step to the next (newer) Line with a Match |

In the Search Bar, type a regex and press `enter`: the regex stays on show, read-only, every Match is highlighted, and the Log View keeps following. Press `up` or `down` to step to the previous or next Line with a Match. The Log View stops following, centres on that Line and marks it with `>`. The first press finds the newest Line with a Match, searching up from the bottom of the Log View. `G` follows again and drops the `>`, but keeps the highlights. A bad regex shows its error in red. Press `/` to edit the regex, `esc` to cancel an edit, and `ctrl+l` to close the Search Bar, end the Search and follow the live logs again.

`ctrl+c` and `ctrl+d` always quit, except while the Prompt is open, and can't be remapped.

## Configuration

Every key above can be changed in `~/.config/logCtrl/keymap.json`. List only the Actions you want to change: each entry is matched by its `action`, and every Action you leave out keeps its default key. `key` and `description` are both optional.

```json
[
  { "action": "quit", "key": "x" },
  { "action": "scroll_up", "key": "pgup" },
  { "action": "scroll_down", "key": "pgdown", "description": "Newer lines" }
]
```

A key is a single character (`x`, `?`) or a key name: `tab`, `esc`, `enter`, `up`, `down`, `pgup`, `pgdown`, `f1` to `f20`, `ctrl+a` to `ctrl+z`, and so on. Add `alt+` in front for Alt combinations, e.g. `alt+x`.

If the file has a mistake, `logctrl` won't start, and it prints an error naming the file. This happens when:

- the file isn't valid JSON, or can't be read
- an entry has no `action`, or an action id that doesn't exist
- a key isn't a single character or a known key name (e.g. `Tab`, `ctrl-x`)
- two Actions end up on the same key
- an Action is put on `ctrl+c` or `ctrl+d`
- `close` or `close_search` is put on a single printable character (it would be typed into the Prompt or the Search Bar instead)

## Roadmap

- Filtering the live logs
- Commands, typed in the Prompt, applied to the live logs

## Development

```bash
make dev     # run from source
make test    # go vet + tests
make cover   # coverage check, as run in CI
make hooks   # enable the pre-commit hook
node tests/tools/mockLogger.js | go run .   # watch a fake log producer (needs Node.js)
```

See `docs/architecture.md` for the architecture, `GLOSSARY.md` for the terms used in the code and this README, and `docs/adr/` for design decisions.

## License

MIT. See [LICENSE](LICENSE).
