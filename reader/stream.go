package reader

import (
	"bufio"
	"io"
	"log"
	"os"
)

const (
	logFileLocation = ""
	logFileName     = "logCtrl_logFile.txt"
)

type Stream interface {
	Start(chan bool)
	SetBufferSize(int)
	GetLive() string
	Close()
}

type stream struct {
	// log files
	logFile *os.File
	logFeed *os.File

	// access buffers
	liveAccessBuffer   Buffer
	randomAccessBuffer Buffer

	// notification channel
	next chan bool
}

// NewStream - Creates a new stream object from the `logFeed` provided.
func NewStream(logFeed *os.File) Stream {
	logFile, err := os.CreateTemp(logFileLocation, logFileName)
	if err != nil {
		log.Fatalf("unable to create temp log file - %v", err)
	}

	return &stream{
		logFile: logFile,
		logFeed: logFeed,
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
			s.next <- true
			s.liveAccessBuffer.Push(liveBuffer.Text())
		}
	}()
}

// SetBufferSize - sets the size of `randomAccessBuffer` and `liveAccessBuffer`.
// This function would create a new buffer entirely and so all previous data
// will be wiped clean.
func (s *stream) SetBufferSize(size int) {
	s.randomAccessBuffer = NewBuffer(size)
	s.liveAccessBuffer = NewBuffer(size)
}

// GetLive - returns the live logs that are currently being pushed
func (s *stream) GetLive() string {
	return s.liveAccessBuffer.Stringify("\n")
}

// Close - closes all pipes and files
func (s *stream) Close() {
	s.logFeed.Close()
	s.logFile.Close()
	close(s.next)
}
