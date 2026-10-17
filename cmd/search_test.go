package cmd_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/internal/testutil"
	"github.com/SpandanBG/logctrl/reader"
)

func Test_SearchMarksMatchesInEachLine(t *testing.T) {
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
		{name: "empty regex ends the Search", regex: "", text: "abc", matches: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			search := cmd.NewSearch()
			if err := search.Set(test.regex); err != nil {
				t.Fatalf("Set(%q) - %v", test.regex, err)
			}

			lines := search.Run([]reader.Line{{Text: test.text}})

			testutil.Equal(t, 1, len(lines))
			testutil.Equal(t, test.text, lines[0].Text)
			testutil.Equal(t, fmt.Sprint(test.matches), fmt.Sprint(lines[0].Matches))
		})
	}
}

func Test_SearchMarksEveryLineInTheWindow(t *testing.T) {
	search := cmd.NewSearch()
	if err := search.Set("err"); err != nil {
		t.Fatal(err)
	}

	lines := search.Run([]reader.Line{{Text: "err 1"}, {Text: "ok"}, {Text: "2 err"}})

	testutil.Equal(t, true, slices.Equal([]reader.Match{{Start: 0, End: 3}}, lines[0].Matches))
	testutil.Equal(t, 0, len(lines[1].Matches))
	testutil.Equal(t, true, slices.Equal([]reader.Match{{Start: 2, End: 5}}, lines[2].Matches))
}

func Test_SearchWithBadRegexMarksNothing(t *testing.T) {
	search := cmd.NewSearch()
	if err := search.Set("err"); err != nil {
		t.Fatal(err)
	}

	if err := search.Set("err("); err == nil {
		t.Fatal("expected an error for a regex that does not compile")
	}

	lines := search.Run([]reader.Line{{Text: "err("}})
	testutil.Equal(t, 0, len(lines[0].Matches))
}
