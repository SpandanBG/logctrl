// Package logview reads the Log View out of a rendered view, for UI tests.
package logview

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/charmbracelet/x/ansi"
)

// Rows - the rows inside the Log View in `view`, each its Gutter cells then
// its Line's shown text, without the border, padding or colors; empty rows
// below the last Line too.
func Rows(view string) []string {
	var rows []string
	for _, row := range inside(ansi.Strip(view)) {
		rows = append(rows, strings.TrimRight(row, " "))
	}
	return rows
}

// ShownLines - the shown text of each Line the Log View in `view` shows,
// without its Gutter cells.
func ShownLines(view string) []string {
	var lines []string
	for _, row := range Rows(view) {
		cells := []rune(row)
		if len(cells) > reader.GutterWidth {
			lines = append(lines, string(cells[reader.GutterWidth:]))
		}
	}
	return lines
}

// Bottom - the shown text of the bottom Line the Log View in `view` shows;
// "" if none.
func Bottom(view string) string {
	lines := ShownLines(view)
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// Marked - the shown text of the Line the Gutter in `view` marks as the
// Current Line; "" if none.
func Marked(view string) string {
	for _, row := range Rows(view) {
		cells := []rune(row)
		if len(cells) >= reader.GutterWidth && strings.ContainsRune(string(cells[:reader.GutterWidth]), '>') {
			return string(cells[reader.GutterWidth:])
		}
	}
	return ""
}

// Highlighted - the text of each Match the Log View in `view` highlights, top
// first, as `[a b]`.
func Highlighted(view string) string {
	var each []string
	for _, row := range inside(view) {
		for _, match := range highlight.FindAllStringSubmatch(row, -1) {
			each = append(each, match[1])
		}
	}
	return fmt.Sprint(each)
}

// highlight - a Match as the Search highlights it.
var highlight = regexp.MustCompile("\x1b\\[30;43m(.*?)\x1b\\[0m")

// ------------------------- Private

// inside - the rows of `view` inside the Log View's border, still padded.
func inside(view string) []string {
	var rows []string
	for _, row := range strings.Split(view, "\n") {
		if inner, ok := strings.CutPrefix(row, "│"); ok {
			rows = append(rows, strings.TrimSuffix(inner, "│"))
		}
	}
	return rows
}
