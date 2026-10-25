# Contributing

How a change gets into logctrl: from an issue, through `/tdd`, to a reviewed commit. The rules the code itself is reviewed against are in [`CODING_STANDARDS.md`](CODING_STANDARDS.md); IDs like `TDD-1` below point there.

## Setup

- **Go**: the version in `go.mod` (1.23 or later). logctrl runs on macOS and Linux; it needs `/dev/tty`.
- **Hooks**: run `make hooks` once. The pre-commit hook in `.githooks/` runs `make test`, so a commit that isn't `gofmt`ed or fails vet or the tests is stopped.
- **Node.js** (optional): only for `make mock`, a fake process that writes log lines, for trying logctrl by hand: `node tests/tools/mockLogger.js | go run main.go`.

## Commands

```bash
make dev     # go run main.go
make build   # binary at bin/out
make test    # gofmt check + go vet + go test -race ./...
make cover   # coverage across packages against COVER_MIN, as CI runs it (never cached)
make hooks   # use the committed pre-commit hook
go test -bench=. ./reader   # benchmarks
scripts/prove-fails.sh <file> <old> <new> <go test args>   # break code, run tests, restore (TDD-4)
scripts/check-dividers.sh [file.go...]                     # CC-9 Private divider (run by make test)
```

## Workflow

1. **Start from an issue.** Work is tracked in [GitHub Issues](https://github.com/SpandanBG/logctrl/issues), managed with `gh` (conventions in [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md)). Open one before starting anything that isn't already filed, with a `bug`, `enhancement` or `documentation` label.
2. **Agree the seams.** Before writing a test, write down the public interface under test and agree it (TDD-3). If the shape of a module is in question, settle it first, and record the outcome on the issue.
3. **Red, green, one slice at a time.** Each behaviour change goes through `/tdd`: one failing test, watched failing for the expected reason, then the least code that passes it, then the next test (TDD-1, TDD-2). A test added for code that already exists is shown to fail by breaking the code (TDD-4).
4. **Refactor at review.** Cleanup happens after green, as its own step (TDD-6, CC-11), not while making a test pass.
5. **Review.** Run `/code-review` against the point the work started from. It checks two axes separately: **Standards** (this file and `CODING_STANDARDS.md`) and **Spec** (the issue). Fix the findings, or say on the issue why not.
6. **Commit**, then close the issue (see below).

## Docs that change with the code

A change isn't done until the docs that describe it agree with it:

| Doc | Update when |
|---|---|
| [`GLOSSARY.md`](GLOSSARY.md) | A domain term is added or sharpened. Definitions only: no implementation details. |
| [`docs/adr/`](docs/adr/) | A decision is hard to reverse, surprising without context, **and** the result of a real trade-off. All three, or no ADR. Number it after the last one and follow the existing ADRs' shape. |
| [`docs/architecture.md`](docs/architecture.md) | Architecture, a package's role, a key interface, or a key binding changes. |
| [`CLAUDE.md`](CLAUDE.md) | A command, a testing rule, or where an agent should look changes. |
| [`README.md`](README.md) | Anything a user sees: a key, an Action id, configuration, install or usage. |

## Commits

- One logical change per commit, passing `make test` (the pre-commit hook checks).
- A short lower-case summary of what changed, as in the history: `fix: truncate long error message in regex bar`, `add gutter for line markings`.
- Say which issue it closes in the body, `Closes #12`, so GitHub closes it on push.
- Don't commit generated or local files (`bin/`, `cover.out`, editor state).

## Pull requests

- CI (`.github/workflows/test.yml`) runs `make test` and `make cover` on every push and pull request; both must pass.
- The description says what behaviour changed, links the issue, and lists any docs updated.
- External pull requests are reviewed as changes, not triaged as feature requests: open an issue for an idea first.
