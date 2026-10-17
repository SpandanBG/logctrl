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
	exec := cmd.NewExecutor(stream)

	testutil.Equal(t, "error 1\nok 2", exec.GetWindow())
	testutil.Equal(t, "[[] []]", matches(exec.Lines()))
}

func Test_ExecutorSearchMarksMatchesInWindowKeepingText(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1\nok 2\n2 errors")
	exec := cmd.NewExecutor(stream)

	if err := exec.Search("err"); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[{0 3}] [] [{2 5}]]", matches(exec.Lines()))
	testutil.Equal(t, "error 1\nok 2\n2 errors", exec.GetWindow())
}

func Test_ExecutorSearchMarksLinesThatComeIntoWindow(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("ok 1")
	exec := cmd.NewExecutor(stream)
	if err := exec.Search("err"); err != nil {
		t.Fatal(err)
	}

	stream.SetWindow("ok 1\nerror 2") // a new line arrived, or the Window scrolled

	testutil.Equal(t, "[[] [{0 3}]]", matches(exec.Lines()))
}

func Test_ExecutorEmptyRegexEndsSearch(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream)
	if err := exec.Search("err"); err != nil {
		t.Fatal(err)
	}

	if err := exec.Search(""); err != nil {
		t.Fatal(err)
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorBadRegexEndsSearchWithError(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error( 1")
	exec := cmd.NewExecutor(stream)
	if err := exec.Search("err"); err != nil {
		t.Fatal(err)
	}

	if err := exec.Search("error("); err == nil {
		t.Fatal("expected an error for a regex that does not compile")
	}

	testutil.Equal(t, "[[]]", matches(exec.Lines()))
}

func Test_ExecutorSearchWhileWindowIsRead(t *testing.T) {
	stream := fakestream.New()
	stream.SetWindow("error 1")
	exec := cmd.NewExecutor(stream)

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
		exec.Search(fmt.Sprint("err", i%2))
	}
	wg.Wait()
}
