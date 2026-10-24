package cmd_test

import (
	"fmt"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/internal/fakestream"
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

// feed - a real reader.Stream over a pipe, `height` lines tall, fed `lines`.
// Returns it with a func feeding it more lines.
func feed(t *testing.T, height int, lines ...string) (reader.Stream, func(...string)) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	next := make(chan bool)
	stream := reader.NewStream(r)
	stream.SetBufferSize(height)
	stream.Start(next)
	t.Cleanup(func() {
		w.Close()
		stream.Close()
	})

	more := func(lines ...string) {
		for _, line := range lines {
			fmt.Fprintln(w, line)
			<-next
		}
	}
	more(lines...)
	return stream, more
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

// enter - searches for `regex` and finds the Current Line, as the UI does on
// enter.
func enter(t *testing.T, exec *cmd.Executor, regex string) {
	t.Helper()
	if err := exec.Update(cmd.SearchFor{Regex: regex}); err != nil {
		t.Fatal(err)
	}
	if err := exec.Update(cmd.FindCurrentLine{}); err != nil {
		t.Fatal(err)
	}
}

// shown - the shown text of the Window, as the UI reads it.
func shown(exec *cmd.Executor) string {
	shown, _ := exec.GetWindow()
	return shown
}

// numbers - the Line Number of each Line.
func numbers(lines []reader.Line) string {
	each := make([]int, 0, len(lines))
	for _, line := range lines {
		each = append(each, line.Number)
	}
	return fmt.Sprint(each)
}

func Test_ExecutorSearchCentresWindowOnNewestLineWithMatchInIt(t *testing.T) {
	stream, more := feed(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	enter(t, exec, "err")
	more("ok 20")

	testutil.Equal(t, "[14 15 16 17 18]", numbers(exec.Lines())) // Pinned: line 20 is not shown
	testutil.Equal(t,
		"ok 14\nok 15\n\x1b[30;43merr\x1b[0m 16\nok 17\nok 18", // not bold: the Gutter marks it
		shown(exec))
}

func Test_ExecutorSearchLooksBackForLineWithMatchWhenWindowHasNone(t *testing.T) {
	// 12,000 lines: the Live Buffer holds the newest 10,000; line 500 is only in
	// the Log File
	stream, _ := feed(t, 5, numbered(12_000, 300, 500)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	enter(t, exec, "err")

	testutil.Equal(t, "[498 499 500 501 502]", numbers(exec.Lines()))
}

func Test_ExecutorSearchPinsWindowWhereItIsWhenNoLineMatches(t *testing.T) {
	stream, more := feed(t, 3, numbered(10)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	enter(t, exec, "err")
	more("ok 10")

	testutil.Equal(t, "ok 7\nok 8\nok 9", shown(exec))
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

func Test_ExecutorSearchForOnlyPinsWindowUntilCurrentLineIsFound(t *testing.T) {
	for _, test := range []struct {
		name     string
		updates  []any
		expected int
	}{
		{name: "SearchFor alone", updates: []any{cmd.SearchFor{Regex: "err"}}, expected: -1},
		{name: "then FindCurrentLine", updates: []any{cmd.SearchFor{Regex: "err"}, cmd.FindCurrentLine{}}, expected: 2},
		{name: "after the Search ended", updates: []any{cmd.SearchFor{Regex: "err"}, cmd.SearchFor{}, cmd.FindCurrentLine{}}, expected: -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream, more := feed(t, 3, numbered(10, 2)...)
			exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
			for _, msg := range test.updates {
				exec.Update(msg)
			}
			more("ok 10")

			testutil.Equal(t, test.expected, currentLine(exec))
		})
	}
}

func Test_ExecutorPreviousMatchStepsToOlderLineWithMatch(t *testing.T) {
	stream, _ := feed(t, 5, numbered(30, 3, 10, 16)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
	stream, more := feed(t, 5, numbered(20, 3, 10)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
		more(expected.more...)
		if err := exec.Update(cmd.NextMatch{}); err != nil {
			t.Fatal(err)
		}
		testutil.Equal(t, expected.window, numbers(exec.Lines()))
		testutil.Equal(t, expected.current, currentLine(exec))
	}
}

func Test_ExecutorNextMatchFindsLineWithMatchArrivedSinceNoneMatched(t *testing.T) {
	stream, more := feed(t, 3, numbered(10)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	enter(t, exec, "err")
	more(numbered(20, 15)[10:]...)

	exec.Update(cmd.PreviousMatch{}) // nothing older to step to
	testutil.Equal(t, -1, currentLine(exec))
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
			stream, more := feed(t, 3, numbered(10, 2)...)
			exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
			for _, msg := range test.updates {
				exec.Update(msg)
			}
			stream.MoveToEnd()

			exec.Update(cmd.PreviousMatch{})
			more(numbered(12, 11)[10:]...)
			exec.Update(cmd.NextMatch{})
			more("ok 12")

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
	stream, _ := feed(t, 5, numbered(3_000, 100, 2_900)...)
	reads := &counted{Stream: stream}
	exec := cmd.NewExecutor(reads, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
	stream, _ := feed(t, 5, numbered(3_000, 100, 200, 2_900)...)
	unread := &cutOff{Stream: stream}
	exec := cmd.NewExecutor(unread, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
	stream, _ := feed(t, 5, numbered(3_000, 100, 2_900)...)
	gate := newGated(stream)
	exec := cmd.NewExecutor(gate, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
	stream, _ := feed(t, 5, numbered(3_000, 100, 2_900, 2_950)...)
	gate := newGated(stream)
	exec := cmd.NewExecutor(gate, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
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
	stream, more := feed(t, 3)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	enter(t, exec, "err")
	more("err 0", "ok 1")

	exec.Update(cmd.NextMatch{})

	testutil.Equal(t, 0, currentLine(exec))
}

func Test_ExecutorShowsStreamWindowWhileNoSearch(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1\nok 2")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	testutil.Equal(t, "error 1\nok 2", shown(exec))
	testutil.Equal(t, "[[] []]", matches(exec.Lines()))
}

func Test_ExecutorSearchMarksMatchesInWindowKeepingText(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1\nok 2\n2 errors")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

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
			stream := fakestream.New()
			stream.SetWindow(test.text)
			exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
			if err := exec.Update(cmd.SearchFor{Regex: test.regex}); err != nil {
				t.Fatalf("SearchFor(%q) - %v", test.regex, err)
			}

			testutil.Equal(t, fmt.Sprint([][]reader.Match{test.matches}), matches(exec.Lines()))
		})
	}
}

func Test_ExecutorSearchMarksLinesThatComeIntoWindow(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("ok 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	stream.SetWindow("ok 1\nerror 2") // a new line arrived, or the Window scrolled

	testutil.Equal(t, "[[] [{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorEmptyRegexEndsSearch(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(cmd.SearchFor{}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorMessageNoPassKnowsChangesNothing(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(struct{ Regex string }{Regex: "1"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorBadRegexEndsSearchWithError(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error( 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	if err := exec.Update(cmd.SearchFor{Regex: "error("}); err == nil {
		t.Fatal("expected an error for a regex that does not compile")
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorSearchWhileWindowIsRead(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	// the UI reads the Window in a tea.Cmd while Update starts a Search
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			exec.GetWindow()
		}
	}()
	for i := range 100 {
		exec.Update(cmd.SearchFor{Regex: fmt.Sprint("err", i%2)})
	}
	wg.Wait()
}

func Test_ExecutorHighlightsEachMatchInWindow(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1, error 2\nok 3")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())

	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t,
		"\x1b[30;43merr\x1b[0mor 1, \x1b[30;43merr\x1b[0mor 2\nok 3",
		shown(exec))
}

// gutter - the Gutter cells of the Window, as the UI reads them.
func gutter(exec *cmd.Executor) string {
	_, gutter := exec.GetWindow()
	return gutter
}

func Test_ExecutorGutterMarksCurrentLineAndMovesWithIt(t *testing.T) {
	stream, _ := feed(t, 5, numbered(20, 12, 16)...)
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()), cmd.NewGutter())
	testutil.Equal(t, "  \n  \n  \n  \n  ", gutter(exec)) // no Search, no mark

	enter(t, exec, "err") // the Window is centred on line 16
	testutil.Equal(t, "  \n  \n >\n  \n  ", gutter(exec))

	exec.MoveBy(-1)
	testutil.Equal(t, "  \n  \n  \n >\n  ", gutter(exec))
}
