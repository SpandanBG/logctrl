package cmd_test

import (
	"fmt"
	"sync"
	"testing"

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

func Test_ExecutorShowsStreamWindowWhileNoSearch(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1\nok 2")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))

	testutil.Equal(t, "error 1\nok 2", exec.GetWindow())
	testutil.Equal(t, "[[] []]", matches(exec.Lines()))
}

func Test_ExecutorSearchMarksMatchesInWindowKeepingText(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1\nok 2\n2 errors")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))

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
			exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))
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
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))
	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	stream.SetWindow("ok 1\nerror 2") // a new line arrived, or the Window scrolled

	testutil.Equal(t, "[[] [{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorEmptyRegexEndsSearch(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))
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
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))
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
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))
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
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))

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
	exec := cmd.NewExecutor(stream, cmd.NewSearch(testutil.Colors()))

	if err := exec.Update(cmd.SearchFor{Regex: "err"}); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t,
		"\x1b[30;43merr\x1b[0mor 1, \x1b[30;43merr\x1b[0mor 2\nok 3",
		exec.GetWindow())
}
