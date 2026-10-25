package headless_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/headless"
	"github.com/SpandanBG/logctrl/internal/logfeed"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/reader"
)

// start - the Logs over the real Executor over a Log Feed, `height` Lines
// tall, fed `lines`.
func start(t *testing.T, height int, lines ...string) (*headless.Logs, *logfeed.Feed) {
	t.Helper()
	return startOver(t, func(stream reader.Stream) reader.Stream { return stream }, height, lines...)
}

// startOver - start, with the Executor over `wrap` of the Log Feed's stream.
func startOver(t *testing.T, wrap func(reader.Stream) reader.Stream, height int, lines ...string) (*headless.Logs, *logfeed.Feed) {
	t.Helper()
	stream, feed := logfeed.New(t)
	logs := headless.New(cmd.NewExecutor(wrap(stream), testutil.Colors()))
	t.Cleanup(logs.Close)
	update(t, logs, cmd.Resize{Height: height})
	arrive(t, feed, lines...)
	return logs, feed
}

// update - hands `msg` to `logs`, failing the test on an error.
func update(t *testing.T, logs *headless.Logs, msg any) {
	t.Helper()
	if err := logs.Update(msg); err != nil {
		t.Fatal(err)
	}
}

// arrive - writes `lines` to the Log Feed and waits for them to arrive.
func arrive(t *testing.T, feed *logfeed.Feed, lines ...string) {
	t.Helper()
	feed.Write(lines...)
	waitFor(t, feed.Arrived)
}

// numbered - lines "ok 0".."ok n-1".
func numbered(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprint("ok ", i)
	}
	return lines
}

func Test_LogsWindowGivesShownTextAndGutterOfEachLine(t *testing.T) {
	logs, _ := start(t, 3, numbered(5)...)

	shown, gutter := logs.Window()

	testutil.Equal(t, "[ok 2 ok 3 ok 4]", fmt.Sprint(shown))
	testutil.Equal(t, `["  " "  " "  "]`, fmt.Sprintf("%q", gutter))
}

// waitFor - waits until `done`, failing the test after a second.
func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("still not done after a second")
		}
		time.Sleep(time.Millisecond)
	}
}

func Test_LogsWindowLeavesOutLinesTheFilterHides(t *testing.T) {
	logs, _ := start(t, 5, "ok 0", "err 1", "ok 2", "err 3", "ok 4")
	update(t, logs, cmd.SearchFor{Regex: "err"})
	update(t, logs, cmd.PreviousMatch{}) // err 3 is the Current Line

	update(t, logs, cmd.FilterFor{Regex: "1|3"})

	shown, gutter := logs.Window()
	testutil.Equal(t, "[\x1b[30;43merr\x1b[0m 1 \x1b[30;43merr\x1b[0m 3]", fmt.Sprint(shown))
	testutil.Equal(t, `["  " " >"]`, fmt.Sprintf("%q", gutter)) // the mark stays beside its Line
}

// notified - whether a notice reaches `logs` within a second, taking it.
func notified(logs *headless.Logs) bool {
	select {
	case <-logs.Changed():
		return true
	case <-time.After(time.Second):
		return false
	}
}

// waiting - whether `logs` has a notice waiting now, taking it.
func waiting(logs *headless.Logs) bool {
	select {
	case <-logs.Changed():
		return true
	default:
		return false
	}
}

// drain - takes the notices left on `logs` by Lines arriving, until none has
// come for a while, as testutil.Runtime settles.
func drain(logs *headless.Logs) {
	for {
		select {
		case <-logs.Changed():
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func Test_LogsNotifiesWhenNewLineArrives(t *testing.T) {
	logs, feed := start(t, 3)
	drain(logs)

	arrive(t, feed, "ok 0")

	testutil.Equal(t, true, notified(logs))
}

func Test_LogsScrollMovesWindowByItsLines(t *testing.T) {
	for _, test := range []struct {
		name     string
		scrolls  []int
		expected string
	}{
		{name: "up", scrolls: []int{-1}, expected: "[ok 6 ok 7 ok 8]"},
		{name: "down", scrolls: []int{-3, 1}, expected: "[ok 5 ok 6 ok 7]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logs, _ := start(t, 3, numbered(10)...)

			for _, n := range test.scrolls {
				update(t, logs, cmd.Scroll{N: n})
			}

			shown, _ := logs.Window()
			testutil.Equal(t, test.expected, fmt.Sprint(shown))
		})
	}
}

func Test_LogsNotifiesAfterEveryUpdate(t *testing.T) {
	for _, test := range []struct {
		name string
		msg  any
	}{
		{name: "Scroll up", msg: cmd.Scroll{N: -1}},
		{name: "Scroll down at the newest Line, moving nothing", msg: cmd.Scroll{N: 1}},
		{name: "Resize", msg: cmd.Resize{Height: 2}},
		{name: "Search", msg: cmd.SearchFor{Regex: "ok"}},
		{name: "Filter", msg: cmd.FilterFor{Regex: "ok"}},
		{name: "Follow", msg: cmd.Follow{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			logs, _ := start(t, 3, numbered(10)...)
			drain(logs)

			update(t, logs, test.msg)

			testutil.Equal(t, true, notified(logs))
		})
	}
}

func Test_LogsResizeMakesWindowThatTall(t *testing.T) {
	logs, _ := start(t, 3, numbered(10)...)

	update(t, logs, cmd.Resize{Height: 2})

	shown, _ := logs.Window()
	testutil.Equal(t, "[ok 8 ok 9]", fmt.Sprint(shown))
}

func Test_LogsFollowDropsCurrentLineAndShowsNewestLines(t *testing.T) {
	logs, _ := start(t, 3, numbered(10)...)
	update(t, logs, cmd.SearchFor{Regex: "ok 2"})
	update(t, logs, cmd.PreviousMatch{}) // Pins the Window on ok 2, the Current Line

	update(t, logs, cmd.Follow{})

	shown, gutter := logs.Window()
	testutil.Equal(t, "[ok 7 ok 8 ok 9]", fmt.Sprint(shown))
	testutil.Equal(t, `["  " "  " "  "]`, fmt.Sprintf("%q", gutter))
}

func Test_LogsCollapsesNoticesNotYetTaken(t *testing.T) {
	logs, _ := start(t, 3) // no Lines, so every notice is left by an Update
	waiting(logs)

	update(t, logs, cmd.Scroll{N: -1})
	update(t, logs, cmd.SearchFor{Regex: "ok"})

	testutil.Equal(t, true, waiting(logs))
	testutil.Equal(t, false, waiting(logs))
}

func Test_LogsKeepsReadingLogFeedWhileNoticesAreNotTaken(t *testing.T) {
	logs, feed := start(t, 3)

	arrive(t, feed, numbered(100)...) // fails unless every Line arrives

	shown, _ := logs.Window()
	testutil.Equal(t, "[ok 97 ok 98 ok 99]", fmt.Sprint(shown))
}

func Test_LogsWindowIsReadWhileSearchStarts(t *testing.T) {
	logs, _ := start(t, 1, "error 1")

	// the UI reads the Window in a tea.Cmd while its Update starts a Search
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			logs.Window()
		}
	}()
	for i := range 100 {
		update(t, logs, cmd.SearchFor{Regex: fmt.Sprint("err", i%2)})
	}
	wg.Wait()

	shown, _ := logs.Window()
	testutil.Equal(t, "[error 1]", fmt.Sprint(shown)) // the last regex, err1, matches nothing
}

func Test_LogsWindowLeavesOutTheMarkOfCurrentLineTheFilterHides(t *testing.T) {
	logs, _ := start(t, 5, "ok 0", "err 1", "ok 2", "err 3", "ok 4")
	update(t, logs, cmd.SearchFor{Regex: "err"})
	update(t, logs, cmd.PreviousMatch{}) // err 3 is the Current Line

	update(t, logs, cmd.FilterFor{Regex: "ok"})

	shown, gutter := logs.Window()
	testutil.Equal(t, "[ok 0 ok 2 ok 4]", fmt.Sprint(shown))
	testutil.Equal(t, `["  " "  " "  "]`, fmt.Sprintf("%q", gutter))
}

// gated - a reader.Stream whose reads outside the Window, once armed, wait
// until released, so a test can act while a step's scan is running.
type gated struct {
	reader.Stream
	armed   atomic.Bool
	reading chan struct{} // receives once per read while armed
	release chan struct{} // closed to let reads finish
}

func (g *gated) Read(from, to int) []reader.Line {
	if g.armed.Load() {
		g.reading <- struct{}{}
		<-g.release
	}
	return g.Stream.Read(from, to)
}

func Test_LogsResizeLeavesStepRunning(t *testing.T) {
	gate := &gated{reading: make(chan struct{}, 100), release: make(chan struct{})}
	lines := numbered(3_000)
	lines[100] = "err 100"
	logs, _ := startOver(t, func(stream reader.Stream) reader.Stream {
		gate.Stream = stream
		return gate
	}, 5, lines...)
	update(t, logs, cmd.SearchFor{Regex: "err"})

	gate.armed.Store(true)
	stepped := make(chan error)
	go func() { stepped <- logs.Update(cmd.PreviousMatch{}) }() // scans back to err 100
	<-gate.reading
	update(t, logs, cmd.Resize{Height: 3}) // as when a bar opens
	gate.armed.Store(false)
	close(gate.release)
	if err := <-stepped; err != nil {
		t.Fatal(err)
	}

	shown, gutter := logs.Window()
	testutil.Equal(t, "[ok 99 \x1b[30;43merr\x1b[0m 100 ok 101]", fmt.Sprint(shown))
	testutil.Equal(t, `["  " " >" "  "]`, fmt.Sprintf("%q", gutter))
}

func Test_LogsChangedClosesOnceClosed(t *testing.T) {
	logs, _ := start(t, 3, numbered(3)...)

	logs.Close()

	waitFor(t, func() bool {
		select {
		case _, open := <-logs.Changed():
			return !open
		default:
			return false
		}
	})
	update(t, logs, cmd.Scroll{N: -1}) // still safe to update, and to Close again
}
