package reader

import (
	"bufio"
	"log"
	"os"
	"syscall"
)

type Source interface {
	Stream() chan string
	Close()
	IsPiped() bool
}

type src struct {
	logStream *os.File
	reader    *bufio.Scanner
	stream    chan string
	open      bool // tells if the stream channel is open.
	isPiped   bool // if `true` the the app has been run as `A | B`.
}

// ResolveSource - get's the source of the log and attaches it to the buffer
func ResolveSource() Source {
	source := &src{
		open:   true,
		stream: make(chan string),
	}

	// verify if app is ran as piped
	if !source.isRanAsPipe() {
		return nil
	}

	// prepare log stream for piped logs
	source.prepareLogStream()

	// start streaming
	go source.startStream()

	return source
}

// Stream - returns the string channel where Stdin is written into.
func (s *src) Stream() chan string {
	return s.stream
}

// IsPiped - returns `true` if app is ran as `A | B` where A is the logger.
func (s *src) IsPiped() bool {
	return s.isPiped
}

// Close - closes stream
func (s *src) Close() {
	if s.open {
		close(s.stream)
	}
}

// --------------------- private methods and function

// startStream - starts reading from `os.Stdin` into `buffer.Reader`.
func (s *src) startStream() {
	// Attach log stream to buffer
	s.reader = bufio.NewScanner(s.logStream)

	// read till `os.Stdin` closes
	for s.reader.Scan() {
		if !s.open {
			break
		}

		s.stream <- s.reader.Text()
	}

	// close stream channel
	close(s.stream)
	s.open = false
}

// prepareLogStream - since `tview` captures the `stdin`, we need to make sure
// `tview` uses `/dev/tty` as f0 & f1, and move `stdin` to next unsued fd (f3).
//
// TODO: use file `CONIN$` for `Windows` platform.
func (s *src) prepareLogStream() {
	logPipe, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("unable to dup stdin: %v", err)
	}

	s.logStream = os.NewFile(uintptr(logPipe), "logPipe")
	if err := syscall.Dup2(0, int(s.logStream.Fd())); err != nil {
		log.Fatalf("unable to dup2 Stdin to LogStream: %v", err.Error())
	}

	ttyIn, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("cannot open /dev/tty as read only: %v", err)
	}
	ttyOut, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		log.Fatalf("cannot open /dev/tty as write only: %v", err)
	}

	if err := syscall.Dup2(int(ttyIn.Fd()), 0); err != nil {
		log.Fatalf("unable to dup2 tty to 0: %v", err.Error())
	}
	if err := syscall.Dup2(int(ttyOut.Fd()), 1); err != nil {
		log.Fatalf("unable to dup2 tty to 1: %v", err.Error())
	}
}

// isRanAsPipe - verifies if the app has be ran as `A | B` where A is the logger.
func (s *src) isRanAsPipe() bool {
	// Get the file info of Stdin
	stat, _ := os.Stdin.Stat()

	// verify if Stdin is from a char device (i.e. executed as a pipe - `./a.out | logctrl`)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		s.isPiped = false
	} else {
		s.isPiped = true
	}

	return s.isPiped
}
