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
}

type stream struct {
	feed    bufio.Scanner
	logFile *os.File
}

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
		feed:    *bufio.NewScanner(logReader),
		logFile: logFile,
	}
}

func (s *stream) All() string {
	data, err := os.ReadFile(s.logFile.Name())
	if err != nil {
		log.Fatalf("unable to read logFile.txt temp file - %v", err)
	}

	return string(data)
}

func (s *stream) Next() string {
	if s.feed.Scan() {
		return s.feed.Text()
	}
	return ""
}

// startStream - copies feed from `fromFeed` into both `toFeed` and `logFile`
func startStream(fromFeed, toFeed, logFile *os.File) {
	teeReader := io.TeeReader(fromFeed, logFile)
	io.Copy(toFeed, teeReader)
}
