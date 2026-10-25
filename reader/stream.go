package reader

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"sync"
)

const (
	logFileLocation = ""
	logFileName     = "logCtrl_logFile.txt"

	// DefaultCapacity - max lines held in the live buffer (see docs/adr/0001)
	DefaultCapacity = 10_000

	// defaultChunkSize - lines per Chunk loaded from the log file (see docs/adr/0004)
	defaultChunkSize = 1_000
)

type Stream interface {
	Start(chan bool)
	SetBufferSize(int)
	Height() int
	Lines() []Line
	Total() int
	First() int
	Read(from, to int) []Line
	MoveToEnd()
	MoveBy(int) int
	PinCentredOn(int)
	Close()
}

type stream struct {
	// log files
	logFile *os.File
	logFeed *os.File

	// display state; `mu` guards both
	mu      sync.Mutex
	history *History
	window  *Window

	// notification channel
	next chan bool
}

// NewStream - Creates a new stream object from the `logFeed` provided.
func NewStream(logFeed *os.File) Stream {
	return newStream(logFeed, DefaultCapacity, defaultChunkSize)
}

// Start - starts the stream. Takes `next` boolean channel which would be
// notified when new logs has been fed into the stream from the producer.
func (s *stream) Start(next chan bool) {
	s.next = next

	go func() {
		teeReader := io.TeeReader(s.logFeed, s.logFile)
		lines := bufio.NewScanner(teeReader)
		lines.Split(scanRawLines)
		for lines.Scan() {
			s.mu.Lock()
			s.history.Push(lines.Text())
			s.mu.Unlock()

			s.next <- true
		}
	}()
}

// SetBufferSize - sets the height of the window. Held lines are kept.
func (s *stream) SetBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window.SetHeight(size)
}

// Height - the height of the window, as set by SetBufferSize.
func (s *stream) Height() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window.Height()
}

// Lines - returns the lines currently in the window.
func (s *stream) Lines() []Line {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window.Lines()
}

// Total - the number of lines received so far.
func (s *stream) Total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.history.Total()
}

// First - the line number of the oldest line that can be read: the first of
// the session, or the oldest held once the Log File has failed to read.
func (s *stream) First() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.history.First()
}

// Read - the lines with line numbers in [from, to), wherever the window is,
// leaving out any the History can no longer show.
func (s *stream) Read(from, to int) []Line {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		from, to := max(from, s.history.First()), min(to, s.history.Total())
		if from >= to {
			return nil
		}
		if lines, ok := s.history.Lines(from, to); ok {
			return lines
		}
	}
}

// MoveToEnd - makes the window follow the newest lines.
func (s *stream) MoveToEnd() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window.MoveToEnd()
}

// MoveBy - scrolls the window by `n` lines (negative is up / older, positive
// is down / newer) and returns the number of lines actually moved.
func (s *stream) MoveBy(n int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.window.MoveBy(n)
}

// PinCentredOn - pins the window with line `number` in its middle, stopping at
// the first and newest lines.
func (s *stream) PinCentredOn(number int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.window.PinCentredOn(number)
}

// Close - closes all pipes and files
func (s *stream) Close() {
	s.logFeed.Close()
	s.logFile.Close()
	close(s.next)
}

// ------------------------- Private

// newStream - NewStream holding `capacity` lines in memory and loading older
// ones in Chunks of `chunkSize` lines.
func newStream(logFeed *os.File, capacity, chunkSize int) Stream {
	logFile, err := os.CreateTemp(logFileLocation, logFileName)
	if err != nil {
		log.Fatalf("unable to create temp log file - %v", err)
	}

	history := NewHistory(capacity, chunkSize, logFile)

	return &stream{
		logFile: logFile,
		logFeed: logFeed,
		history: history,
		window:  NewWindow(history),
	}
}

// scanRawLines - like bufio.ScanLines but keeps the terminator, so the History
// can tell where each line starts in the log file.
func scanRawLines(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i+1], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
