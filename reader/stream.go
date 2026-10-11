package reader

import (
	"bufio"
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
	mu         sync.Mutex
	liveBuffer *LiveBuffer
	window     *Window

	// notification channel
	next chan bool
}

// NewStream - Creates a new stream object from the `logFeed` provided.
func NewStream(logFeed *os.File) Stream {
	logFile, err := os.CreateTemp(logFileLocation, logFileName)
	if err != nil {
		log.Fatalf("unable to create temp log file - %v", err)
	}

	liveBuffer := NewLiveBuffer(defaultCapacity)

	return &stream{
		logFile:    logFile,
		logFeed:    logFeed,
		liveBuffer: liveBuffer,
		window:     NewWindow(liveBuffer),
	}
}

// Start - starts the stream. Takes `next` boolean channel which would be
// notified when new logs has been fed into the stream from the producer.
func (s *stream) Start(next chan bool) {
	s.next = next

	go func() {
		teeReader := io.TeeReader(s.logFeed, s.logFile)
		liveBuffer := bufio.NewScanner(teeReader)
		for liveBuffer.Scan() {
			s.mu.Lock()
			s.liveBuffer.Push(liveBuffer.Text())
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

// GetWindow - returns the lines currently in the window joined by newlines.
func (s *stream) GetWindow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.window.Lines(), "\n")
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

// Close - closes all pipes and files
func (s *stream) Close() {
	s.logFeed.Close()
	s.logFile.Close()
	close(s.next)
}
