package cmd_test

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/internal/logfeed"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/reader"
)

// matches - the Matches of each Line, one Line per entry.
func matches(lines []reader.Line) string {
	each := make([][]reader.Match, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Matches)
	}
	return fmt.Sprint(each)
}

// numbered - lines "ok 0".."ok n-1", but "err i" for each i in `errs`.
func numbered(n int, errs ...int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprint("ok ", i)
		if slices.Contains(errs, i) {
			lines[i] = fmt.Sprint("err ", i)
		}
	}
	return lines
}

// enter - searches for `regex` and finds the Current Line, as the user does
// with enter then a first step.
func enter(t *testing.T, exec *cmd.Executor, regex string) {
	t.Helper()
	if err := exec.Update(cmd.SearchFor{Regex: regex}); err != nil {
		t.Fatal(err)
	}
	if err := exec.Update(cmd.PreviousMatch{}); err != nil {
		t.Fatal(err)
	}
}

// shown - the shown text of each Line in the Window the Filter does not
// hide, joined by newlines.
func shown(exec *cmd.Executor) string {
	var each []string
	for _, line := range exec.Lines() {
		if !line.Filter.Hide {
			each = append(each, line.Shown)
		}
	}
	return strings.Join(each, "\n")
}

// numbers - the Line Number of each Line.
func numbers(lines []reader.Line) string {
	each := make([]int, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Number)
	}
	return fmt.Sprint(each)
}

func Test_ExecutorSearchForKeepsWindowFollowingWithNoCurrentLine(t *testing.T) {
	stream, feed := logfeed.Started(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())

	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}
	feed.Write("ok 20")

	testutil.Equal(t, "[16 17 18 19 20]", numbers(exec.Lines()))
	testutil.Equal(t, -1, currentLine(exec))
	testutil.Equal(t, "[[{0 3}] [] [] [] []]", matches(exec.Lines()))
}

func Test_ExecutorFirstStepCentresWindowOnNewestLineWithMatchAtOrAboveBottom(t *testing.T) {
	for _, test := range []struct {
		name string
		step any
	}{
		{name: "PreviousMatch", step: cmd.PreviousMatch{}},
		{name: "NextMatch", step: cmd.NextMatch{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, feed := logfeed.Started(t, 5, numbered(20, 12, 16)...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
				t.Fatal(err)
			}
			feed.Write("ok 20", "ok 21")

			if err := exec.Update(test.step); err != nil {
				t.Fatal(err)
			}
			feed.Write("ok 22")

			testutil.Equal(t, "[14 15 16 17 18]", numbers(exec.Lines())) // Pinned
			testutil.Equal(t, 16, currentLine(exec))
		})
	}
}

func Test_ExecutorFollowDropsCurrentLineKeepingMatches(t *testing.T) {
	stream, feed := logfeed.Started(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	testutil.Equal(t, 16, currentLine(exec))

	if err := exec.Update(cmd.Follow{}); err != nil {
		t.Fatal(err)
	}
	stream.MoveToEnd() // as the Log View does
	feed.Write("ok 20")
	testutil.Equal(t, -1, currentLine(exec))
	testutil.Equal(t, "[[{0 3}] [] [] [] []]", matches(exec.Lines()))

	exec.Update(cmd.PreviousMatch{}) // finds the Current Line again
	testutil.Equal(t, 16, currentLine(exec))
}

func Test_ExecutorSearchCentresWindowOnNewestLineWithMatchInIt(t *testing.T) {
	stream, feed := logfeed.Started(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())

	enter(t, exec, "err")
	feed.Write("ok 20")

	testutil.Equal(t, "[14 15 16 17 18]", numbers(exec.Lines())) // Pinned: line 20 is not shown
	testutil.Equal(t,
		"ok 14\nok 15\n\x1b[30;43merr\x1b[0m 16\nok 17\nok 18", // not bold: the Gutter marks it
		shown(exec))
}

func Test_ExecutorSearchLooksBackForLineWithMatchWhenWindowHasNone(t *testing.T) {
	// 12,000 lines: the Live Buffer holds the newest 10,000; line 500 is only in
	// the Log File
	stream, _ := logfeed.Started(t, 5, numbered(12_000, 300, 500)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())

	enter(t, exec, "err")

	testutil.Equal(t, "[498 499 500 501 502]", numbers(exec.Lines()))
}

func Test_ExecutorSearchKeepsWindowFollowingWhenNoLineMatches(t *testing.T) {
	stream, feed := logfeed.Started(t, 3, numbered(10)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())

	enter(t, exec, "err")
	feed.Write("ok 10")

	testutil.Equal(t, "ok 8\nok 9\nok 10", shown(exec))
	testutil.Equal(t, "  \n  \n  ", gutter(exec)) // no Current Line
}

// currentLine - the Line Number of the Line the Gutter marks, or -1 if none
// is.
func currentLine(exec *cmd.Executor) int {
	for _, line := range exec.Lines() {
		if line.Gutter[1] == '>' {
			return line.Number
		}
	}
	return -1
}

func Test_ExecutorPreviousMatchStepsToOlderLineWithMatch(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(30, 3, 10, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "r") // "err" has two Matches
	testutil.Equal(t, 16, currentLine(exec))

	for _, expected := range []struct {
		current int
		window  string
	}{
		{current: 10, window: "[8 9 10 11 12]"},
		{current: 3, window: "[1 2 3 4 5]"},
		{current: 3, window: "[1 2 3 4 5]"}, // the oldest: nothing to step to
	} {
		if err := exec.Update(cmd.PreviousMatch{}); err != nil {
			t.Fatal(err)
		}
		testutil.Equal(t, expected.window, numbers(exec.Lines()))
		testutil.Equal(t, expected.current, currentLine(exec))
	}
}

func Test_ExecutorNextMatchStepsToNewerLineWithMatchIncludingLinesArrivedSince(t *testing.T) {
	stream, feed := logfeed.Started(t, 5, numbered(20, 3, 10)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	exec.Update(cmd.PreviousMatch{})
	testutil.Equal(t, 3, currentLine(exec))

	for _, expected := range []struct {
		more    []string
		current int
		window  string
	}{
		{current: 10, window: "[8 9 10 11 12]"},
		{current: 10, window: "[8 9 10 11 12]"}, // the newest: nothing to step to
		{more: numbered(30, 25)[20:], current: 25, window: "[23 24 25 26 27]"},
	} {
		feed.Write(expected.more...)
		if err := exec.Update(cmd.NextMatch{}); err != nil {
			t.Fatal(err)
		}
		testutil.Equal(t, expected.window, numbers(exec.Lines()))
		testutil.Equal(t, expected.current, currentLine(exec))
	}
}

func Test_ExecutorFirstStepFindsLineWithMatchArrivedSinceNoneMatched(t *testing.T) {
	stream, feed := logfeed.Started(t, 3, numbered(10)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	feed.Write(numbered(20, 15)[10:]...)

	exec.Update(cmd.NextMatch{})

	testutil.Equal(t, "[14 15 16]", numbers(exec.Lines()))
	testutil.Equal(t, 15, currentLine(exec))
}

func Test_ExecutorStepsNowhereWhileNoSearch(t *testing.T) {
	for _, test := range []struct {
		name    string
		updates []any
	}{
		{name: "never searched"},
		{name: "search ended", updates: []any{cmd.SearchFor{Regex: "err"}, cmd.SearchFor{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, feed := logfeed.Started(t, 3, numbered(10, 2)...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			for _, msg := range test.updates {
				exec.Update(msg)
			}
			stream.MoveToEnd()

			exec.Update(cmd.PreviousMatch{})
			feed.Write(numbered(12, 11)[10:]...)
			exec.Update(cmd.NextMatch{})
			feed.Write("ok 12")

			testutil.Equal(t, "ok 10\nerr 11\nok 12", shown(exec)) // still Following
		})
	}
}

// counted - a reader.Stream counting the reads made outside the Window.
type counted struct {
	reader.Stream
	reads int
}

func (c *counted) Read(from, to int) []reader.Line {
	c.reads++
	return c.Stream.Read(from, to)
}

func Test_ExecutorRemembersLinesWithMatchFoundUntilPassesAreUpdated(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(3_000, 100, 2_900)...)
	reads := &counted{Stream: stream}
	exec := cmd.NewExecutor(reads, testutil.Colors())
	enter(t, exec, "err")
	exec.Update(cmd.PreviousMatch{})
	testutil.Equal(t, 100, currentLine(exec))
	scanned := reads.reads

	exec.Update(cmd.NextMatch{})
	exec.Update(cmd.PreviousMatch{})
	testutil.Equal(t, 100, currentLine(exec))
	testutil.Equal(t, scanned, reads.reads) // stepped over ground already scanned

	exec.Update(struct{}{}) // any message to the Passes
	exec.Update(cmd.NextMatch{})
	testutil.Equal(t, 2_900, currentLine(exec))
	testutil.Equal(t, true, reads.reads > scanned)
}

// cutOff - a reader.Stream whose reads leave out Lines older than `first`, as
// once the Log File fails to read (see docs/adr/0004).
type cutOff struct {
	reader.Stream
	first int
}

func (c *cutOff) Read(from, to int) []reader.Line {
	return c.Stream.Read(max(from, c.first), to)
}

func (c *cutOff) First() int {
	return c.first
}

func Test_ExecutorDropsCurrentLineOnceItCannotBeRead(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(3_000, 100, 200, 2_900)...)
	unread := &cutOff{Stream: stream}
	exec := cmd.NewExecutor(unread, testutil.Colors())
	enter(t, exec, "err")
	exec.Update(cmd.PreviousMatch{})
	exec.Update(cmd.PreviousMatch{})
	exec.Update(cmd.NextMatch{})
	testutil.Equal(t, 200, currentLine(exec))

	unread.first = 250
	exec.Update(cmd.PreviousMatch{}) // line 100 can no longer be read
	testutil.Equal(t, "[198 199 200 201 202]", numbers(exec.Lines()))
	exec.Update(cmd.NextMatch{})
	testutil.Equal(t, 2_900, currentLine(exec))
}

// gated - a reader.Stream whose reads outside the Window, once armed, wait
// until released, so a test can act while a scan is running.
type gated struct {
	reader.Stream
	armed   atomic.Bool
	reading chan struct{} // receives once per read while armed
	release chan struct{} // closed to let reads finish
}

func newGated(stream reader.Stream) *gated {
	return &gated{Stream: stream, reading: make(chan struct{}, 100), release: make(chan struct{})}
}

func (g *gated) Read(from, to int) []reader.Line {
	if g.armed.Load() {
		g.reading <- struct{}{}
		<-g.release
	}
	return g.Stream.Read(from, to)
}

// within - fails the test unless `do` returns within a second.
func within(t *testing.T, what string, do func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		do()
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("%s did not return while a scan was running", what)
	}
}

func Test_ExecutorSearchForCancelsStepInFlight(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(3_000, 100, 2_900)...)
	gate := newGated(stream)
	exec := cmd.NewExecutor(gate, testutil.Colors())
	enter(t, exec, "err")

	gate.armed.Store(true)
	stepped := make(chan struct{})
	go func() {
		defer close(stepped)
		exec.Update(cmd.PreviousMatch{})
	}()
	<-gate.reading

	within(t, "SearchFor", func() { exec.Update(cmd.SearchFor{}) })
	gate.armed.Store(false)
	close(gate.release)
	<-stepped

	stream.MoveToEnd()
	testutil.Equal(t, "[2995 2996 2997 2998 2999]", numbers(exec.Lines())) // the step was not applied
	testutil.Equal(t, -1, currentLine(exec))
}

func Test_ExecutorIgnoresStepWhileScanRuns(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(3_000, 100, 2_900, 2_950)...)
	gate := newGated(stream)
	exec := cmd.NewExecutor(gate, testutil.Colors())
	enter(t, exec, "err")
	exec.Update(cmd.PreviousMatch{}) // to 2900, remembered
	testutil.Equal(t, 2_900, currentLine(exec))

	gate.armed.Store(true)
	stepped := make(chan struct{})
	go func() {
		defer close(stepped)
		exec.Update(cmd.PreviousMatch{}) // scans for 100
	}()
	<-gate.reading

	within(t, "NextMatch", func() { exec.Update(cmd.NextMatch{}) })
	testutil.Equal(t, 2_900, currentLine(exec)) // not 2950
	gate.armed.Store(false)
	close(gate.release)
	<-stepped

	testutil.Equal(t, 100, currentLine(exec))
}

func Test_ExecutorNextMatchFindsFirstLineWhenSearchStartedBeforeAnyArrived(t *testing.T) {
	stream, feed := logfeed.Started(t, 3)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	feed.Write("err 0", "ok 1")

	exec.Update(cmd.NextMatch{})

	testutil.Equal(t, 0, currentLine(exec))
}

func Test_ExecutorShowsStreamWindowWhileNoSearch(t *testing.T) {
	stream, _ := logfeed.Started(t, 2, "error 1", "ok 2")
	exec := cmd.NewExecutor(stream, testutil.Colors())

	testutil.Equal(t, "error 1\nok 2", shown(exec))
	testutil.Equal(t, "[[] []]", matches(exec.Lines()))
}

func Test_ExecutorSearchMarksMatchesInWindowKeepingText(t *testing.T) {
	stream, _ := logfeed.Started(t, 3, "error 1", "ok 2", "2 errors")
	exec := cmd.NewExecutor(stream, testutil.Colors())

	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	lines := exec.Lines()
	testutil.Equal(t, "[[{0 3}] [] [{2 5}]]", matches(lines))
	testutil.Equal(t, "2 errors", lines[2].Text)
}

func Test_ExecutorSearchMarksEachMatchOfRegex(t *testing.T) {
	tests := []struct {
		name    string
		regex   string
		text    string
		matches []reader.Match
	}{
		{name: "one match", regex: "err", text: "error: disk full", matches: []reader.Match{{Start: 0, End: 3}}},
		{name: "several matches", regex: `\d+`, text: "a1 b22 c333", matches: []reader.Match{{Start: 1, End: 2}, {Start: 4, End: 6}, {Start: 8, End: 11}}},
		{name: "no match", regex: "warn", text: "error: disk full", matches: nil},
		{name: "case-sensitive as typed", regex: "ERR", text: "error", matches: nil},
		{name: "case-insensitive when asked", regex: "(?i)ERR", text: "error", matches: []reader.Match{{Start: 0, End: 3}}},
		{name: "offsets are in bytes", regex: "llo", text: "héllo", matches: []reader.Match{{Start: 3, End: 6}}},
		{name: "empty matches are dropped", regex: "x*", text: "axxb", matches: []reader.Match{{Start: 1, End: 3}}},
		{name: "only empty matches is no match", regex: "^", text: "abc", matches: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stream, _ := logfeed.Started(t, 2, strings.Split(test.text, "\n")...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			if err := exec.Update(cmd.SearchFor{Regex: test.regex}); err != nil {
				t.Fatalf("SearchFor(%q) - %v", test.regex, err)
			}

			testutil.Equal(t, fmt.Sprint([][]reader.Match{test.matches}), matches(exec.Lines()))
		})
	}
}

func Test_ExecutorSearchMarksLinesThatComeIntoWindow(t *testing.T) {
	stream, feed := logfeed.Started(t, 2, "ok 1")
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	feed.Write("error 2") // a new line arrived

	testutil.Equal(t, "[[] [{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorEmptyRegexEndsSearch(t *testing.T) {
	stream, _ := logfeed.Started(t, 1, "error 1")
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(cmd.SearchFor{}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorMessageNoPassKnowsChangesNothing(t *testing.T) {
	stream, _ := logfeed.Started(t, 1, "error 1")
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(struct{ Regex string }{Regex: "1"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorBadRegexEndsSearchWithError(t *testing.T) {
	stream, _ := logfeed.Started(t, 1, "error( 1")
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(cmd.SearchFor{Regex: "error("}); err == nil {
		t.Fatal("expected an error for a regex that does not compile")
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorHighlightsEachMatchInWindow(t *testing.T) {
	stream, _ := logfeed.Started(t, 2, "error 1, error 2", "ok 3")
	exec := cmd.NewExecutor(stream, testutil.Colors())

	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t,
		"\x1b[30;43merr\x1b[0mor 1, \x1b[30;43merr\x1b[0mor 2\nok 3",
		shown(exec))
}

// hidden - whether the Filter hides each Line, one Line per entry.
func hidden(lines []reader.Line) string {
	each := make([]bool, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Filter.Hide)
	}
	return fmt.Sprint(each)
}

func Test_ExecutorFilterHidesLinesWithoutMatchKeepingThemInWindow(t *testing.T) {
	stream, _ := logfeed.Started(t, 3, "error 1", "ok 2", "2 errors")
	exec := cmd.NewExecutor(stream, testutil.Colors())

	if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[false true false]", hidden(exec.Lines()))
}

func Test_ExecutorFilterShowsEachLineItsRegexMatchesAnywhere(t *testing.T) {
	for _, test := range []struct {
		name   string
		regex  string
		text   string
		hidden string
	}{
		{name: "case-sensitive as typed", regex: "ERR", text: "error\nERROR", hidden: "[true false]"},
		{name: "case-insensitive when asked", regex: "(?i)ERR", text: "error\nok", hidden: "[false true]"},
		{name: "an empty match counts", regex: "^", text: "error\nok", hidden: "[false false]"},
		{name: "empty regex ends the Filter", regex: "", text: "error\nok", hidden: "[false false]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, _ := logfeed.Started(t, 2, strings.Split(test.text, "\n")...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
				t.Fatal(err)
			}

			if err := exec.Update(cmd.FilterFor{Regex: test.regex}); err != nil {
				t.Fatalf("FilterFor(%q) - %v", test.regex, err)
			}

			testutil.Equal(t, test.hidden, hidden(exec.Lines()))
		})
	}
}

// masked - a Pass replacing the shown text of every Line.
type masked struct{}

func (masked) Update(msg any) error { return nil }

func (masked) Run(lines []reader.Line) []reader.Line {
	for i := range lines {
		lines[i].Shown = "***"
	}
	return lines
}

func Test_ExecutorFilterMatchesEachLineAsItCameFromLogFeed(t *testing.T) {
	stream, _ := logfeed.Started(t, 2, "error 1", "ok 2")
	exec := cmd.NewExecutor(stream, testutil.Colors(), masked{})

	if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[false true]", hidden(exec.Lines()))
}

func Test_ExecutorFilterKeepsWindowMode(t *testing.T) {
	for _, test := range []struct {
		name   string
		pinned bool // Pin the Window by stepping to a Current Line first
		window string
	}{
		{name: "Following stays Following", window: "[17 18 19 20 21]"},
		{name: "Pinned stays on its Line Numbers", pinned: true, window: "[14 15 16 17 18]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, feed := logfeed.Started(t, 5, numbered(20, 12, 16)...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			if test.pinned {
				enter(t, exec, "err") // centred on line 16
			}

			for _, regex := range []string{"ok", ""} { // entering, then ending, a Filter
				if err := exec.Update(cmd.FilterFor{Regex: regex}); err != nil {
					t.Fatal(err)
				}
			}
			feed.Write("ok 20", "ok 21")

			testutil.Equal(t, test.window, numbers(exec.Lines()))
		})
	}
}

func Test_ExecutorBadRegexEndsFilterWithError(t *testing.T) {
	stream, _ := logfeed.Started(t, 2, "error( 1", "ok 2")
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(cmd.FilterFor{Regex: "error("}); err == nil {
		t.Fatal("expected an error for a regex that does not compile")
	}

	testutil.Equal(t, "[false false]", hidden(exec.Lines()))
}

func Test_ExecutorStepsPassOverLinesTheFilterHides(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(30, 3, 10, 16, 22)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	if err := exec.Update(cmd.FilterFor{Regex: "[^0-9](3|16)$"}); err != nil { // hides lines 10 and 22
		t.Fatal(err)
	}

	enter(t, exec, "err") // passes over line 22
	testutil.Equal(t, 16, currentLine(exec))

	for _, expected := range []struct {
		step    any
		current int
	}{
		{step: cmd.PreviousMatch{}, current: 3}, // passes over line 10
		{step: cmd.NextMatch{}, current: 16},
		{step: cmd.NextMatch{}, current: 16}, // line 22 is hidden: nothing to step to
	} {
		if err := exec.Update(expected.step); err != nil {
			t.Fatal(err)
		}
		testutil.Equal(t, expected.current, currentLine(exec))
	}
}

func Test_ExecutorKeepsCurrentLineTheFilterHides(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(20, 3, 10, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	testutil.Equal(t, 16, currentLine(exec))

	if err := exec.Update(cmd.FilterFor{Regex: "1[47]"}); err != nil { // hides line 16
		t.Fatal(err)
	}
	testutil.Equal(t, 16, currentLine(exec))
	testutil.Equal(t, "ok 14\nok 17", shown(exec))

	if err := exec.Update(cmd.FilterFor{}); err != nil {
		t.Fatal(err)
	}
	testutil.Equal(t, "ok 14\nok 15\n\x1b[30;43merr\x1b[0m 16\nok 17\nok 18", shown(exec))
}

func Test_ExecutorStepsOnFromCurrentLineTheFilterHides(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(20, 3, 10, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	enter(t, exec, "err")
	if err := exec.Update(cmd.FilterFor{Regex: "[^0-9](3|10)$"}); err != nil { // hides line 16
		t.Fatal(err)
	}

	if err := exec.Update(cmd.PreviousMatch{}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, 10, currentLine(exec))
}

// gutter - the Gutter cells of each Line in the Window, an empty cell as a
// space, joined by newlines.
func gutter(exec *cmd.Executor) string {
	var each []string
	for _, line := range exec.Lines() {
		each = append(each, strings.ReplaceAll(string(line.Gutter[:]), "\x00", " "))
	}
	return strings.Join(each, "\n")
}

func Test_ExecutorGutterMarksCurrentLineAndMovesWithIt(t *testing.T) {
	stream, _ := logfeed.Started(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	testutil.Equal(t, "  \n  \n  \n  \n  ", gutter(exec)) // no Search, no mark

	enter(t, exec, "err") // the Window is centred on line 16
	testutil.Equal(t, "  \n  \n >\n  \n  ", gutter(exec))

	exec.MoveBy(-1)
	testutil.Equal(t, "  \n  \n  \n >\n  ", gutter(exec))
}

func Test_ExecutorWindowTakesInOlderLinesWhileFilterHidesLines(t *testing.T) {
	stream, _ := logfeed.Started(t, 3, numbered(20, 11, 13, 15, 17, 19)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())

	if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}
	testutil.Equal(t, "err 15\nerr 17\nerr 19", shown(exec))
	testutil.Equal(t, "[15 16 17 18 19]", numbers(exec.Lines()))

	if err := exec.Update(cmd.FilterFor{}); err != nil {
		t.Fatal(err)
	}
	testutil.Equal(t, "err 17\nok 18\nerr 19", shown(exec)) // the Window as it was
}

func Test_ExecutorWindowTakesInNewerLinesOnceAtFirstLineWhileFilterHidesLines(t *testing.T) {
	stream, _ := logfeed.Started(t, 3, numbered(20, 1, 5, 9, 13)...)
	exec := cmd.NewExecutor(stream, testutil.Colors())
	exec.MoveBy(-20) // Pinned on the first Lines, 0 to 2

	if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "err 1\nerr 5\nerr 9", shown(exec))
	testutil.Equal(t, "[0 1 2 3 4 5 6 7 8 9]", numbers(exec.Lines()))
}

func Test_ExecutorWindowTakesInAtMostCapacityLinesBeyondIt(t *testing.T) {
	for _, test := range []struct {
		name   string
		scroll int // from Following on Lines Capacity+1 to Capacity+3
		err    int // the Line Number of the only Line the Filter shows
		shown  string
	}{
		{name: "the Capacity-th older Line is taken in", err: 1, shown: "err 1"},
		{name: "an older Line further back is not", err: 0, shown: ""},
		{name: "the Capacity-th newer Line is taken in", scroll: -reader.DefaultCapacity - 4, err: reader.DefaultCapacity + 2, shown: fmt.Sprint("err ", reader.DefaultCapacity+2)},
		{name: "a newer Line further on is not", scroll: -reader.DefaultCapacity - 4, err: reader.DefaultCapacity + 3, shown: ""},
		// Pinned on Lines 5 to 7, the 5 older Lines leave Capacity-5 newer ones
		{name: "both ways, the last newer Line in the Capacity is taken in", scroll: -reader.DefaultCapacity + 4, err: reader.DefaultCapacity + 2, shown: fmt.Sprint("err ", reader.DefaultCapacity+2)},
		{name: "both ways, a newer Line past the Capacity is not", scroll: -reader.DefaultCapacity + 4, err: reader.DefaultCapacity + 3, shown: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, _ := logfeed.Started(t, 3, numbered(reader.DefaultCapacity+4, test.err)...)
			exec := cmd.NewExecutor(stream, testutil.Colors())
			exec.MoveBy(test.scroll)

			if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
				t.Fatal(err)
			}

			testutil.Equal(t, test.shown, shown(exec))
		})
	}
}

// unreadable - a reader.Stream whose reads outside the Window find nothing, as
// when the Log File fails to read before First has moved up.
type unreadable struct {
	reader.Stream
}

func (unreadable) Read(from, to int) []reader.Line {
	return nil
}

func Test_ExecutorWindowStopsTakingInLinesOnceNoneCanBeRead(t *testing.T) {
	for _, test := range []struct {
		name   string
		scroll int
	}{
		{name: "older Lines, while Following", scroll: 0},
		{name: "newer Lines, Pinned on the first Lines", scroll: -20},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, _ := logfeed.Started(t, 3, numbered(20, 10)...)
			exec := cmd.NewExecutor(unreadable{stream}, testutil.Colors())
			exec.MoveBy(test.scroll)
			if err := exec.Update(cmd.FilterFor{Regex: "err"}); err != nil {
				t.Fatal(err)
			}

			within(t, "Lines", func() { testutil.Equal(t, "", shown(exec)) })
		})
	}
}
