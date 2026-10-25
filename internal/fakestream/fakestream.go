// Package fakestream provides an in-memory cmd.Stream for UI tests.
package fakestream

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/cmd"
	"github.com/SpandanBG/logctrl/reader"
)

// Stream - a cmd.Stream whose Window text is set by the test.
type Stream struct {
	mu      sync.Mutex
	height  int
	window  string
	gutter  string
	next    chan bool
	closed  bool
	scrolls []int
	follows int
	updates []any
	pins    []int
}

var _ cmd.Stream = (*Stream)(nil)

func New() *Stream {
	return &Stream{}
}

func (s *Stream) Start(next chan bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next = next
}

func (s *Stream) SetBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.height = size
}

func (s *Stream) GetWindow() (shown, gutter string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window, s.gutter
}

// Lines - the Window text, one Line per row, numbered from 0.
func (s *Stream) Lines() []reader.Line {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.window == "" {
		return nil
	}

	var lines []reader.Line
	for i, text := range strings.Split(s.window, "\n") {
		lines = append(lines, reader.Line{Number: i, Text: text})
	}
	return lines
}

// Total - the number of Lines in the Window text.
func (s *Stream) Total() int {
	return len(s.Lines())
}

// First - always 0: the Window text can always be read.
func (s *Stream) First() int {
	return 0
}

// Read - the Lines of the Window text with line numbers in [from, to).
func (s *Stream) Read(from, to int) []reader.Line {
	lines := s.Lines()
	return lines[max(from, 0):min(to, len(lines))]
}

func (s *Stream) MoveToEnd() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.follows++
}

func (s *Stream) MoveBy(n int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scrolls = append(s.scrolls, n)
	return n
}

// Update - records `msg`; returns the error if it is a SearchFor or a
// FilterFor whose regex does not compile.
func (s *Stream) Update(msg any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, msg)
	switch msg := msg.(type) {
	case cmd.SearchFor:
		_, err := regexp.Compile(msg.Regex)
		return err
	case cmd.FilterFor:
		_, err := regexp.Compile(msg.Regex)
		return err
	}
	return nil
}

// PinCentredOn - records `number` as a Pin the UI made.
func (s *Stream) PinCentredOn(number int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pins = append(s.pins, number)
}

func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// Height - the last Window height the UI asked for.
func (s *Stream) Height() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.height
}

// Scrolls - every MoveBy the UI made, in order.
func (s *Stream) Scrolls() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]int(nil), s.scrolls...)
}

// Follows - how many times the UI made the Window Following.
func (s *Stream) Follows() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.follows
}

// Updates - every message the UI handed to the Passes, in order.
func (s *Stream) Updates() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]any(nil), s.updates...)
}

// SetWindow - changes the Window text without notifying the UI, as a Scroll
// does.
func (s *Stream) SetWindow(window string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window = window
}

// SetGutter - changes the Gutter cells GetWindow returns with the Window text.
func (s *Stream) SetGutter(gutter string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gutter = gutter
}

// Closed - whether Close has been called.
func (s *Stream) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Notify - sets the Window text and notifies the UI that new lines arrived.
// Fails the test if the UI is not listening.
func (s *Stream) Notify(t *testing.T, window string) {
	t.Helper()
	s.mu.Lock()
	s.window = window
	next := s.next
	s.mu.Unlock()

	select {
	case next <- true:
	case <-time.After(time.Second):
		t.Fatal("UI is not listening for new lines")
	}
}
