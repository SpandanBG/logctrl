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
	All() string
	Next() string
	Close()
}

type stream struct {
	feed      bufio.Scanner
	logFile   *os.File
	logFeed   *os.File
	logReader *os.File
}

// NewStream - Creates a new stream interface of log data
// from producer to consumer. All logs are saved temporarily
// to a temporary log file. The original feed is multiplexed
// into the log file and into a stream for the ui consumer.
func NewStream(logFeed *os.File) Stream {
	logFile, err := os.CreateTemp(logFileLocation, logFileName)
	if err != nil {
		log.Fatalf("unable to create temp log file - %v", err)
	}

	logReader, logWritter, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create log pipe for stream - %v", err)
	}

	go func() {
		startStream(logFeed, logWritter, logFile)
		defer logReader.Close()
	}()

	return &stream{
		feed:      *bufio.NewScanner(logReader),
		logFile:   logFile,
		logFeed:   logFeed,
		logReader: logReader,
	}
}

// All - reads the entire temp `logFile.txt` and returns the value.
func (s *stream) All() string {
	data, err := os.ReadFile(s.logFile.Name())
	if err != nil {
		log.Fatalf("unable to read logFile.txt temp file - %v", err)
	}

	return string(data)
}

// Next - fetches the next line that has been multiplexed into the stream.
func (s *stream) Next() string {
	if s.feed.Scan() {
		return s.feed.Text()
	}
	return ""
}

// Close - closes all pipes, readers, writers and files.
func (s *stream) Close() {
	s.logFile.Close()
	s.logFeed.Close()
	s.logReader.Close()
}

// startStream - copies feed from `fromFeed` into both `toFeed` and `logFile`
func startStream(fromFeed, toFeed, logFile *os.File) {
	teeReader := io.TeeReader(fromFeed, logFile)
	io.Copy(toFeed, teeReader)
}
