# Coding Standards

The rules every change to logctrl is reviewed against. Each rule has an ID to cite in a review (`GO-7`, `TDD-3`) and a source tag saying where it comes from.

## How to read this

- **Scope.** The rules apply to the code a change adds or touches. Existing code that breaks a rule is listed under [Known exceptions](#known-exceptions), and isn't flagged in a diff that doesn't touch it.
- **House overrides upstream.** A rule tagged **house** is how this repo does it, even where the upstream guide says otherwise; the rule names the upstream rule it overrides, so a review doesn't flag it.
- **Tooling first.** Anything `make test` already catches (`gofmt`, `go vet`) is not repeated here.
- **Domain and design.** Names use the terms in [`GLOSSARY.md`](GLOSSARY.md); decisions in [`docs/adr/`](docs/adr/) bind. A diff may add an `Amended (#N):` paragraph or Consequence to an ADR when it refines the decision without reversing it, naming the issue that changed it; a decision reversed or replaced gets a new ADR that supersedes the old one.
- **Workflow** (issues, commits, docs to update) is in [`CONTRIBUTING.md`](CONTRIBUTING.md).

Source tags:

| Tag | Source |
|---|---|
| CRC | [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) |
| EG | [Effective Go](https://go.dev/doc/effective_go) |
| GSG | [Google Go Style Guide](https://google.github.io/styleguide/go/) and its [Decisions](https://google.github.io/styleguide/go/decisions) |
| CC | Robert C. Martin, _Clean Code_ ([summary](https://gist.github.com/wojteklu/73c6914cc446146b8b533c0988cf8d29)) |
| BECK | Kent Beck, [Canon TDD](https://newsletter.kentbeck.com/p/canon-tdd) |
| FOWLER | Martin Fowler, [Test Driven Development](https://martinfowler.com/bliki/TestDrivenDevelopment.html) |
| CLIG | [Command Line Interface Guidelines](https://clig.dev/) |
| house | This repo's own convention |

## GO: Go

### Formatting and names

- **GO-1** Imports are grouped standard library first, then everything else. _(CRC, EG)_
- **GO-2** Names are `MixedCaps` / `mixedCaps`, never underscores (test function names excepted, see TDD-12). Initialisms keep one case: `URL`, `ID`, `urlPath`, never `Url`. _(CRC, GSG)_
- **GO-3** Package names are short, lower case, one word, and say what the package holds. No `util`, `common`, `misc`, `helper`, `types`. Exported names don't repeat the package: `widget.New`, not `widget.NewWidget`. _(CRC, EG, GSG)_
- **GO-4** A name's length grows with its scope: `i`, `r`, `s` in a few lines; a descriptive name where it's used far from its declaration. _(CRC, GSG)_
- **GO-5** Receiver names are one or two letters abbreviating the type, the same in every method of that type. Never `this` or `self`. _(CRC, GSG)_
- **GO-6** No `Get` prefix on getters: `Owner()`, `SetOwner()`. _(EG, GSG)_
- **GO-7** Imports aren't renamed, except to avoid a collision or for the aliases already in use (`tea` for bubbletea, `ui` for `ui/utils`). No dot imports. _(CRC; aliases are house)_

### Comments

- **GO-8** _house_ Every top-level declaration, exported or not, has a doc comment in the form `name - what it is or does.`, as in `// NewRegexBar - the regex bar ...`. Struct fields say what they hold in a trailing comment. This overrides the upstream `Name does ...` sentence form (CRC "Comment Sentences", GSG).
- **GO-9** Package comments sit directly above the `package` clause, with no blank line. _(CRC, GSG)_

### Errors

- **GO-10** `error` is the last return value. Every error is handled: returned, acted on, or (only in `main`) fatal. Never discarded with `_` or by ignoring the result, unless a comment says why it's safe. _(CRC, EG, GSG)_
- **GO-11** Error strings are lower case with no trailing punctuation, since they get wrapped. Add context by wrapping with `%w`, naming what failed: `fmt.Errorf("keymap %s: %w", path, err)`. _(CRC, EG, GSG)_
- **GO-12** Handle the error first and return early, so the normal path runs down the left edge. No `else` after a `return`. _(CRC, EG, GSG)_
- **GO-13** No in-band errors (`-1`, `""`, `nil` meaning "not found"). Return an extra `ok bool` or an `error`. _(CRC, GSG)_
- **GO-14** No `panic` for normal errors. A `panic` is only for an impossible state, or setup that cannot continue. _(CRC, EG, GSG)_

### Types, interfaces and concurrency

- **GO-15** An interface is defined in the package that uses it, holds only the methods that package calls, and appears once a second adapter exists (as `cmd.Pass`, used by the `Executor`, has the Filter and the Executor tests' `masked`). Functions accept interfaces and return concrete types. _(CRC, GSG)_
- **GO-16** Choose value or pointer receivers deliberately: pointer when a method mutates the receiver or the type holds a lock; value for small immutable types and for `tea.Model`s, which bubbletea copies (house). Don't pass pointers just to save bytes. _(CRC, EG)_
- **GO-17** Every goroutine has a clear exit: what stops it, and when. Prefer channels to coordinate; a mutex around shared state is fine where it's simpler. _(CRC, EG)_
- **GO-18** A `defer` that releases a resource sits right after the line that acquires it. _(EG)_
- **GO-19** An empty slice is declared `var s []T` (nil), not `[]T{}`, unless it's encoded as JSON. _(CRC)_
- **GO-20** Prefer synchronous functions that return their result; let the caller add concurrency. _(CRC)_

## CC: Clean code

- **CC-1** A function does one thing, at one level of abstraction, and is short enough to see whole. _(CC)_
- **CC-2** Few parameters, and no flag (bool) parameters that switch behaviour: split into two functions instead. _(CC)_
- **CC-3** Names reveal intent and can be searched for. A number with meaning is a named constant, or has a comment right beside it saying what it is (`- 1 // leaving a cell for the cursor`). _(CC)_
- **CC-4** Comments say why, or warn of a consequence. They don't repeat what the code says. No commented-out code. _(CC)_
- **CC-5** Name an intermediate result rather than nesting expressions: an explanatory variable is cheaper than a comment. _(CC)_
- **CC-6** Talk to your direct collaborators only (Law of Demeter). Don't walk `a.B().C().D()` through another module's insides. _(CC)_
- **CC-7** The same logic shape isn't written twice in a change. But don't abstract before the second real use: duplication is a signal to consider, not a directive. _(CC, BECK)_
- **CC-8** No speculative generality: no parameters, hooks, options or abstractions the current issue doesn't need. _(CC, BECK)_
- **CC-9** Related code sits together, read top-down: callers above callees. _house_: in a file, the public code comes first and the private helpers last, after a `// ------------------------- Private` divider (exported funcs and methods, `main` and `init` above it; checked by `scripts/check-dividers.sh` in `make test`). _(CC)_
- **CC-10** Prefer a small type to a primitive standing in for a domain concept (`Bar` rather than an `int` or a string). _(CC)_
- **CC-11** Leave the code cleaner than you found it, but in a refactoring step of its own, never mixed into making a test pass (see TDD-6). _(CC, BECK)_

## TDD: Tests and test-driven development

### The loop

- **TDD-1** No production change without a failing test first. Write the test, watch it fail for the expected reason, then write only enough code to pass it. _(BECK, FOWLER)_
- **TDD-2** One vertical slice at a time: one test, then its code, then the next test. Never write all the tests up front. _(BECK)_
- **TDD-3** Test at seams: a package's public interface, agreed before the test is written. Never test private functions, mock internal collaborators, or check through a side channel. Files with no behaviour of their own (constants, empty stubs, `main.go`) get no tests. _(house)_
- **TDD-4** A test added for code that already exists must be shown to fail: break the code on purpose, watch the test fail, undo the break (`scripts/prove-fails.sh`). _(house)_
- **TDD-5** Expected values come from an independent source: a known-good literal, a worked example, the spec. Never paste the code's own output in as the expected value, and never recompute it the way the code does. _(BECK)_
- **TDD-6** Refactoring is its own step, after green (at review), never mixed into making a test pass. _(BECK, FOWLER)_
- **TDD-7** Every test asserts something. Never delete or weaken an assertion to make a test pass. _(BECK)_

### Writing tests

- **TDD-8** When cases vary only by data, the test is table-driven: named struct fields, one `t.Run` subtest per case. _(GSG; house in CLAUDE.md)_
- **TDD-9** A failure message says what was being done, what came back and what was expected, so it can be read without opening the test. _(CRC "Useful Test Failures", GSG)_
- **TDD-10** Test helpers call `t.Helper()`. Prefer `t.Error`, so one run shows every failure; use `t.Fatal` only when the checks after it would be meaningless. _(GSG)_
- **TDD-11** _house_ Assert with the shared helpers in `internal/`: `testutil.Equal(t, expected, actual)` (expected first), `testutil.Runtime` to drive a `tea.Model`, `testutil.Colors` to compare styled output against literals, `logfeed` for a real `reader.Stream` fed over a pipe (tests check what is shown, never which calls were made), and `logview` to read the Log View out of a rendered view. This overrides GSG's "no assertion libraries" and its got-before-want order.
- **TDD-12** _house_ Test functions are named `Test_<Subject><Behaviour>` in the terms of `GLOSSARY.md`, as in `Test_StreamPinCentredOnLine`. GSG allows underscores in test names.
- **TDD-13** Tests are fast, independent and repeatable, and pass under `-race`. They never touch the real `~/.config`, the network, or wall-clock timing beyond `testutil.Runtime`'s settle wait. _(CC; house)_
- **TDD-14** Coverage, counted across packages by `make cover`, never drops below `COVER_MIN` in the `Makefile`. Raise `COVER_MIN` as coverage grows; never lower it to make a change pass. _(house)_

## CLI: Command-line and terminal UI

logctrl is a terminal UI that reads a Log Feed on stdin, so CLIG's guidance applies to its process and its terminal; the TUI's own conventions are house.

- **CLI-1** Exit `0` on success and non-zero on failure. _(CLIG)_
- **CLI-2** Errors go to stderr, written for a person: what went wrong, and what to do about it (name the file, the key, the Action). _(CLIG)_
- **CLI-3** Validate input early, before anything changes: a bad user Keymap stops logctrl before the terminal goes raw. _(CLIG)_
- **CLI-4** Every exit path, error paths included, leaves the terminal as it found it (raw mode restored, alt screen left). Nothing that can exit (`log.Fatal`, `os.Exit`) runs once the terminal is raw without restoring it first. _(CLIG "Signals")_
- **CLI-5** `^C` and `^D` always Quit while the Prompt is closed, and stay outside the Keymap (Reserved). Quitting stops the process piped in. _(CLIG; house)_
- **CLI-6** Every style is built from the lipgloss renderer injected into the module, never `lipgloss.NewStyle()` or a package-level style. That renderer honours `NO_COLOR` and `TERM=dumb`, and lets tests force colours. Use colour sparingly, and red only for errors. _(CLIG; house)_
- **CLI-7** _house_ The UI never matches a literal key. `ui/layout.go` turns a keypress into an Action from the merged Keymap once, and modules get Action-level messages. A new key is a new Action in `config/keymap.json` (ADR 0003).
- **CLI-8** User configuration lives under the XDG config directory, is only read, and is never written without the user asking. _(CLIG)_
- **CLI-9** Environment variable names are UPPERCASE letters, digits and underscores. _(CLIG)_
- **CLI-10** `-h` and `--help` show help and exit. Started with stdin on a terminal (nothing piped in), logctrl shows how to use it rather than waiting. _(CLIG)_
- **CLI-11** The UI answers within about 100 ms. Never block the tea event loop: slow work (reads of the History, scans) runs in a `tea.Cmd` or a goroutine with a clear exit. _(CLIG; GO-17)_
- **CLI-12** When state changes, show it: the Toolbar and the regex bars say which mode logctrl is in and which keys act. _(CLIG)_

## Known exceptions

Existing code that breaks a rule above. Don't flag these in a diff that doesn't touch them; a diff that does should fix them or leave them no worse.

| Rule | Where | What |
|---|---|---|
| GO-3 | `utils/`, `ui/utils/` | package names say nothing about what they hold |
| GO-15 | `cmd.Stream` | one adapter, the `Executor`, since #7 deleted `fakestream`; defined in `cmd`, not in `headless`, which uses it |
| GO-8 | `main.go` (`ChildEnvVar`, `resizePty`, `startDataPump`) | missing or non-house doc comments |
| CLI-9 | `main.go` `ChildEnvVar = "logctrl_child"` | lower-case environment variable ([#14](https://github.com/SpandanBG/logctrl/issues/14)) |
| CLI-10 | `main.go` | no `-h` / `--help`; with nothing piped in, it waits on stdin ([#13](https://github.com/SpandanBG/logctrl/issues/13)) |
| CLI-8 | `main.go` `loadKeymap` | `~/.config/logCtrl` is hard-coded; `XDG_CONFIG_HOME` is ignored ([#14](https://github.com/SpandanBG/logctrl/issues/14)) |
| CLI-4 | `main.go` `startPTY`, `startDataPump`, `resizePty` | `log.Fatalf` after the terminal goes raw exits without restoring it ([#12](https://github.com/SpandanBG/logctrl/issues/12)) |
| GO-10 | `main.go` `startDataPump` | `io.Copy` errors are dropped ([#14](https://github.com/SpandanBG/logctrl/issues/14)) |
| CLI-2 | `main.go` | fatal errors carry `log`'s timestamp prefix; "unable to to open /dev/tty" ([#14](https://github.com/SpandanBG/logctrl/issues/14)) |
| CLI-6 | `ui/components/logView.go` `logViewStyle`, `toolbar.go` `toolbarStyle`, `help.go` `helpStyle` and `helpKeyStyle` | package-level styles with colours, not built from the injected renderer ([#15](https://github.com/SpandanBG/logctrl/issues/15)) |
