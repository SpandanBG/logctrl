package cmd_test

import (
	"fmt"
	"testing"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/internal/logfeed"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/reader"
)

// update - hands `msg` to `exec`, failing the test on an error.
func update(t *testing.T, exec *cmd.Executor, msg any) {
	t.Helper()
	if err := exec.Update(msg); err != nil {
		t.Fatal(err)
	}
}

// start - an Executor over a Log Feed, its Window `height` Lines tall, fed
// `lines`.
func start(t *testing.T, height int, lines ...string) (*cmd.Executor, *logfeed.Feed) {
	t.Helper()
	return startOver(t, func(stream reader.Stream) reader.Stream { return stream }, height, lines...)
}

// startOver - start, with the Executor over `wrap` of the Log Feed's stream.
func startOver(t *testing.T, wrap func(reader.Stream) reader.Stream, height int, lines ...string) (*cmd.Executor, *logfeed.Feed) {
	t.Helper()
	stream, feed := logfeed.Started(t, lines...)
	exec := cmd.NewExecutor(wrap(stream), testutil.Colors())
	update(t, exec, cmd.Resize{Height: height})
	return exec, feed
}

func Test_ExecutorResizeMakesWindowThatTall(t *testing.T) {
	exec, _ := start(t, 5, logfeed.Numbered(10)...)

	update(t, exec, cmd.Resize{Height: 3})

	testutil.Equal(t, "[7 8 9]", numbers(exec.Lines()))
}

func Test_ExecutorWindowFollowsNewestLines(t *testing.T) {
	for _, test := range []struct {
		name     string
		lines    int
		expected string
	}{
		{name: "no Lines yet", lines: 0, expected: "[]"},
		{name: "fewer Lines than its height", lines: 2, expected: "[0 1]"},
		{name: "the newest Lines", lines: 7, expected: "[4 5 6]"},
		{name: "once the Live Buffer is full", lines: reader.DefaultCapacity + 4, expected: fmt.Sprint([]int{reader.DefaultCapacity + 1, reader.DefaultCapacity + 2, reader.DefaultCapacity + 3})},
	} {
		t.Run(test.name, func(t *testing.T) {
			exec, _ := start(t, 3, logfeed.Numbered(test.lines)...)

			testutil.Equal(t, test.expected, numbers(exec.Lines()))
		})
	}
}

func Test_ExecutorScrollMovesWindowAndReachesPasses(t *testing.T) {
	exec, _ := start(t, 5, logfeed.Numbered(20, 12, 16)...)
	enter(t, exec, "err") // centred on line 16

	update(t, exec, cmd.Scroll{N: -2})

	testutil.Equal(t, "[12 13 14 15 16]", numbers(exec.Lines()))
	update(t, exec, cmd.PreviousMatch{}) // the Search forgot what it found: scans again
	testutil.Equal(t, 12, currentLine(exec))
}

func Test_ExecutorScrollPinsWindowUntilItReachesNewestLine(t *testing.T) {
	for _, test := range []struct {
		name     string
		lines    int
		scrolls  []int
		arrive   int // Lines arriving after the Scrolls
		expected string
	}{
		{name: "up Pins", lines: 10, scrolls: []int{-2}, arrive: 5, expected: "[5 6 7]"},
		{name: "down while Following stays Following", lines: 10, scrolls: []int{3}, arrive: 1, expected: "[8 9 10]"},
		{name: "up stops at the first Line", lines: 10, scrolls: []int{-50}, expected: "[0 1 2]"},
		{name: "down while Pinned stays Pinned", lines: 10, scrolls: []int{-5, 2}, arrive: 1, expected: "[4 5 6]"},
		{name: "down to the newest Line Follows", lines: 10, scrolls: []int{-5, 9}, arrive: 5, expected: "[12 13 14]"},
		{name: "fewer Lines than its height stays Following", lines: 2, scrolls: []int{-1}, arrive: 1, expected: "[0 1 2]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			exec, feed := start(t, 3, logfeed.Numbered(test.lines)...)

			for _, n := range test.scrolls {
				update(t, exec, cmd.Scroll{N: n})
			}
			feed.Write(logfeed.Numbered(test.lines + test.arrive)[test.lines:]...)

			testutil.Equal(t, test.expected, numbers(exec.Lines()))
		})
	}
}

func Test_ExecutorPinnedWindowKeepsItsLinesOnceTheyLeaveLiveBuffer(t *testing.T) {
	exec, feed := start(t, 3, logfeed.Numbered(5)...)
	update(t, exec, cmd.Scroll{N: -50}) // Pinned on the first Lines, 0 to 2

	feed.Write(logfeed.Numbered(reader.DefaultCapacity + 5)[5:]...) // 0 to 4 leave the Live Buffer

	testutil.Equal(t, "[0 1 2]", numbers(exec.Lines()))
	update(t, exec, cmd.Scroll{N: 1_000}) // into the next Chunk
	testutil.Equal(t, "[1000 1001 1002]", numbers(exec.Lines()))
}

func Test_ExecutorWindowCentresOnCurrentLine(t *testing.T) {
	for _, test := range []struct {
		name     string
		height   int
		current  int
		expected string
	}{
		{name: "in the middle", height: 3, current: 10, expected: "[9 10 11]"},
		{name: "upper middle when its height is even", height: 4, current: 10, expected: "[9 10 11 12]"},
		{name: "stops at the first Line", height: 3, current: 0, expected: "[0 1 2]"},
		{name: "stops at the newest Line", height: 3, current: 19, expected: "[17 18 19]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			exec, feed := start(t, test.height, logfeed.Numbered(20, test.current)...)

			enter(t, exec, "err")
			feed.Write("ok 20")

			testutil.Equal(t, test.expected, numbers(exec.Lines())) // Pinned: line 20 is not shown
		})
	}
}

func Test_ExecutorFollowMakesWindowFollowing(t *testing.T) {
	exec, feed := start(t, 5, logfeed.Numbered(20, 12, 16)...)
	enter(t, exec, "err") // Pinned, centred on line 16

	update(t, exec, cmd.Follow{})
	feed.Write("ok 20")

	testutil.Equal(t, "[16 17 18 19 20]", numbers(exec.Lines()))
}

// cutOff - a reader.Stream whose Lines older than `first` can no longer be
// read, as once the Log File fails to read: like the History's, its First
// moves up only once a read reaches them (see docs/adr/0004).
type cutOff struct {
	reader.Stream      // the stream read from
	first         int  // the Line Number of the oldest Line that can still be read
	failed        bool // a read has reached Lines older than `first`
}

// Read - the Lines in [from, to), leaving out any older than `first`.
func (c *cutOff) Read(from, to int) []reader.Line {
	if from < c.first {
		c.failed = true
	}
	return c.Stream.Read(max(from, c.first), to)
}

// First - `first`, once a read has reached Lines older than it.
func (c *cutOff) First() int {
	if c.failed {
		return c.first
	}
	return c.Stream.First()
}

func Test_ExecutorWindowMovesUpToFirstLineThatCanBeRead(t *testing.T) {
	for _, test := range []struct {
		name     string
		before   []int // Scrolls before the first Line that can be read moves up to 5
		after    []int // Scrolls after
		expected string
	}{
		{name: "Scroll up stops at it", after: []int{-50}, expected: "[5 6 7]"},
		{name: "Pinned on Lines that can no longer be read", before: []int{-50}, expected: "[5 6 7]"},
		{name: "Scroll down from them", before: []int{-50}, after: []int{1}, expected: "[6 7 8]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			unread := &cutOff{}
			exec, _ := startOver(t, func(stream reader.Stream) reader.Stream {
				unread.Stream = stream
				return unread
			}, 3, logfeed.Numbered(20)...)
			for _, n := range test.before {
				update(t, exec, cmd.Scroll{N: n})
			}

			unread.first = 5
			for _, n := range test.after {
				update(t, exec, cmd.Scroll{N: n})
			}

			testutil.Equal(t, test.expected, numbers(exec.Lines()))
		})
	}
}

func Test_ExecutorWindowMovedUpToNewestLineFollows(t *testing.T) {
	unread := &cutOff{}
	exec, feed := startOver(t, func(stream reader.Stream) reader.Stream {
		unread.Stream = stream
		return unread
	}, 3, logfeed.Numbered(20)...)
	update(t, exec, cmd.Scroll{N: -50}) // Pinned on the first Lines, 0 to 2

	unread.first = 18
	exec.Lines() // moves up to 18, the newest Lines
	feed.Write("ok 20", "ok 21")

	testutil.Equal(t, "[19 20 21]", numbers(exec.Lines()))
}

// arriving - a reader.Stream whose newest Line arrives just after its Total is
// first read, as when a Line arrives while the Window is read.
type arriving struct {
	reader.Stream      // the stream read from
	read          bool // Total has been read
}

// Total - one Line short the first time, as the newest is still arriving.
func (a *arriving) Total() int {
	if !a.read {
		a.read = true
		return a.Stream.Total() - 1
	}
	return a.Stream.Total()
}

func Test_ExecutorFollowingWindowIsAsTallAsItsHeightWhileLineArrives(t *testing.T) {
	stream, _ := logfeed.Started(t, logfeed.Numbered(10)...)
	exec := cmd.NewExecutor(&arriving{Stream: stream}, testutil.Colors())
	update(t, exec, cmd.Resize{Height: 3})

	testutil.Equal(t, "[6 7 8]", numbers(exec.Lines()))
}

func Test_ExecutorScrollMovesWindowByLinesTheFilterDoesNotHide(t *testing.T) {
	exec, _ := start(t, 3, logfeed.Numbered(30, 2, 5, 9, 14, 20, 25, 28)...)
	update(t, exec, cmd.FilterFor{Regex: "err"})

	for _, expected := range []struct {
		scroll int
		shown  string
	}{
		{scroll: -1, shown: "err 14\nerr 20\nerr 25"},
		{scroll: -2, shown: "err 5\nerr 9\nerr 14"},
		{scroll: 1, shown: "err 9\nerr 14\nerr 20"},
	} {
		update(t, exec, cmd.Scroll{N: expected.scroll})
		testutil.Equal(t, expected.shown, shown(exec))
	}
}

func Test_ExecutorScrollDownToNewestLineTheFilterDoesNotHideFollows(t *testing.T) {
	exec, feed := start(t, 3, logfeed.Numbered(30, 5, 9, 14, 20)...) // Lines 21 to 29 are hidden
	update(t, exec, cmd.FilterFor{Regex: "err"})
	update(t, exec, cmd.Scroll{N: -1})
	testutil.Equal(t, "err 5\nerr 9\nerr 14", shown(exec))

	update(t, exec, cmd.Scroll{N: 1})
	feed.Write("err 30")

	testutil.Equal(t, "err 14\nerr 20\nerr 30", shown(exec))
}

func Test_ExecutorScrollPassesOverAtMostCapacityHiddenLines(t *testing.T) {
	newest := reader.DefaultCapacity + 9
	exec, _ := start(t, 3, logfeed.Numbered(newest+1, 0, newest)...)
	update(t, exec, cmd.FilterFor{Regex: "err"})

	update(t, exec, cmd.Scroll{N: -1}) // passes over Capacity hidden Lines, finding none shown
	testutil.Equal(t, fmt.Sprint("err ", newest), shown(exec))

	update(t, exec, cmd.Scroll{N: -1})
	testutil.Equal(t, "err 0", shown(exec))
}

func Test_ExecutorPinnedWindowWhoseTopLineTheFilterHidesShowsFromNextShownLine(t *testing.T) {
	exec, _ := start(t, 3, logfeed.Numbered(30)...)
	update(t, exec, cmd.Scroll{N: -17}) // Pinned on line 10

	update(t, exec, cmd.FilterFor{Regex: "[13]$"}) // hides line 10, but not lines 1 and 3 above it
	testutil.Equal(t, "ok 11\nok 13\nok 21", shown(exec))

	update(t, exec, cmd.FilterFor{})
	testutil.Equal(t, "ok 10\nok 11\nok 12", shown(exec))
}

func Test_ExecutorWindowCentresOnCurrentLineAmongLinesTheFilterDoesNotHide(t *testing.T) {
	for _, test := range []struct {
		name     string
		height   int
		current  int
		expected string
	}{
		{name: "in the middle", height: 3, current: 12, expected: "[8 12 16]"},
		{name: "upper middle when its height is even", height: 4, current: 12, expected: "[8 12 16 20]"},
		{name: "stops at the first shown Line", height: 3, current: 4, expected: "[4 8 12]"},
		{name: "stops at the newest shown Line", height: 3, current: 28, expected: "[20 24 28]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			exec, feed := start(t, test.height, logfeed.Numbered(30, 4, 8, 12, 16, 20, 24, 28)...)
			update(t, exec, cmd.FilterFor{Regex: "err"})

			enter(t, exec, fmt.Sprintf("^err %d$", test.current))
			feed.Write("err 30")

			testutil.Equal(t, test.expected, shownNumbers(exec)) // Pinned: line 30 is not shown
		})
	}
}

func Test_ExecutorWindowCentresOnCurrentLineLookingBackAtMostCapacityLines(t *testing.T) {
	current := reader.DefaultCapacity + 5
	exec, _ := start(t, 3, logfeed.Numbered(current+10, 0, current, current+1)...)
	update(t, exec, cmd.FilterFor{Regex: "err"})

	enter(t, exec, fmt.Sprintf("^err %d$", current)) // line 0 is further back than Capacity Lines

	testutil.Equal(t, fmt.Sprint([]int{current, current + 1}), shownNumbers(exec))
}

func Test_ExecutorWindowMovedUpToNewestLineTheFilterDoesNotHideFollows(t *testing.T) {
	unread := &cutOff{}
	exec, feed := startOver(t, func(stream reader.Stream) reader.Stream {
		unread.Stream = stream
		return unread
	}, 3, logfeed.Numbered(20, 15, 16)...) // Lines 17 to 19 are hidden
	update(t, exec, cmd.Scroll{N: -50}) // Pinned on the first Lines
	update(t, exec, cmd.FilterFor{Regex: "err"})

	unread.first = 15
	exec.Lines() // moves up to 15: its bottom is the newest shown Line
	feed.Write("err 20", "err 21")

	testutil.Equal(t, "[16 20 21]", shownNumbers(exec))
}

// counted - a reader.Stream counting its reads.
type counted struct {
	reader.Stream     // the stream read from
	reads         int // the reads so far
}

// Read - the Lines in [from, to), counting the read.
func (c *counted) Read(from, to int) []reader.Line {
	c.reads++
	return c.Stream.Read(from, to)
}

func Test_ExecutorScrollPastManyHiddenLinesReadsInFewBatches(t *testing.T) {
	reads := &counted{}
	newest := reader.DefaultCapacity + 9
	exec, _ := startOver(t, func(stream reader.Stream) reader.Stream {
		reads.Stream = stream
		return reads
	}, 3, logfeed.Numbered(newest+1, 0, newest)...)
	update(t, exec, cmd.FilterFor{Regex: "err"})

	reads.reads = 0
	update(t, exec, cmd.Scroll{N: -1}) // passes over Capacity hidden Lines

	if reads.reads > 50 {
		t.Fatalf("a Scroll read the History %d times, expected at most 50", reads.reads)
	}
}
