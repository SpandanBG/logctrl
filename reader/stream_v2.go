package reader

import (
	"bufio"
	"io"
	"log"
	"os"
)

type StreamV2 interface {
	Start(chan bool)
	SetBufferSize(int)
	GetLive() string
	Close()
}

type streamV2 struct {
	// log files
	logFile *os.File
	logFeed *os.File

	// access buffers
	liveAccessBuffer   Buffer
	randomAccessBuffer Buffer

	// notification channel
	next chan bool
}

// NewStreamV2 - Creates a new stream object from the `logFeed` provided.
func NewStreamV2(logFeed *os.File) StreamV2 {
	logFile, err := os.CreateTemp(logFileLocation, logFileName)
	if err != nil {
		log.Fatalf("unable to create temp log file (stream-v2) - %v", err)
	}

	return &streamV2{
		logFile: logFile,
		logFeed: logFeed,
	}
}

// Start - starts the stream. Takes `next` boolean channel which would be
// notified when new logs has been fed into the stream from the producer.
func (s *streamV2) Start(next chan bool) {
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
func (s *streamV2) SetBufferSize(size int) {
	s.randomAccessBuffer = NewBuffer(size)
	s.liveAccessBuffer = NewBuffer(size)
}

// GetLive - returns the live logs that are currently being pushed
func (s *streamV2) GetLive() string {
	return s.liveAccessBuffer.Stringify("\n")
}

// Close - closes all pipes and files
func (s *streamV2) Close() {
	s.logFeed.Close()
	s.logFile.Close()
	close(s.next)
}
