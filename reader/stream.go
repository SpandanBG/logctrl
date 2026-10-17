package reader

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	logFileLocation = ""
	logFileName     = "logCtrl_logFile.txt"

	// defaultCapacity - max lines held in the live buffer (see docs/adr/0001)
	defaultCapacity = 10_000

	// defaultChunkSize - lines per Chunk loaded from the log file (see docs/adr/0004)
	defaultChunkSize = 1_000
)

type Stream interface {
	Start(chan bool)
	SetBufferSize(int)
	GetWindow() string
	MoveToEnd()
	MoveBy(int) int
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
	return newStream(logFeed, defaultCapacity, defaultChunkSize)
}

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

// GetWindow - returns the text of the lines currently in the window joined by
// newlines.
func (s *stream) GetWindow() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	lines := s.window.Lines()
	texts := make([]string, 0, len(lines))
	for _, line := range lines {
		texts = append(texts, line.Text)
	}
	return strings.Join(texts, "\n")
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

// Close - closes all pipes and files
func (s *stream) Close() {
	s.logFeed.Close()
	s.logFile.Close()
	close(s.next)
}
