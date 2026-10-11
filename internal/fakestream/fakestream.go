// Package fakestream provides an in-memory reader.Stream for UI tests.
package fakestream

import (
	"sync"
	"testing"
	"time"

	"github.com/SpandanBG/logctrl/reader"
)

// Stream - a reader.Stream whose Window text is set by the test.
type Stream struct {
	mu      sync.Mutex
	height  int
	window  string
	next    chan bool
	closed  bool
	scrolls []int
	follows int
}

var _ reader.Stream = (*Stream)(nil)

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

func (s *Stream) GetWindow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window
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

// SetWindow - changes the Window text without notifying the UI, as a Scroll
// does.
func (s *Stream) SetWindow(window string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window = window
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
