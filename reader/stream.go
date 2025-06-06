package reader

import (
	"bufio"
	"log"
	"os"
)

type Source interface {
	Stream() chan string
	Close()
}

type src struct {
	logStream *os.File
	reader    *bufio.Scanner
	stream    chan string
	open      bool // tells if the stream channel is open
}

// ResolveSource - get's the source of the log and attaches it to the buffer
func ResolveSource() Source {
	source := &src{
		open:   true,
		stream: make(chan string),
	}

	// Get the file info of Stdin
	stat, _ := os.Stdin.Stat()

	// verify if Stdin is from a char device (i.e. executed as a pipe - `./a.out | logctrl`)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil
	}

	// prepare log stream
	source.prepareLogStream()

	// Attach log stream to buffer
	source.reader = bufio.NewScanner(source.logStream)
	return source
}

func (s *src) Stream() chan string {
	go func() {
		for s.reader.Scan() {
			s.stream <- s.reader.Text()
		}
		close(s.stream)
		s.open = false
	}()

	return s.stream
}

// Close - closes stream
func (s *src) Close() {
	if s.open {
		close(s.stream)
	}

	os.Stdin = s.logStream
}

// --------------------- private methods and function

// prepareLogStream - since `tview` captures the `stdin`, we need to make sure
// `tview` uses `/dev/tty` as the `stdin` to unblock programs that have piped
// into `stdin`.
//
// TODO: use file `CONIN$` for `Windows` platform.
func (s *src) prepareLogStream() {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("cannot open /dev/tty: %v", err)
	}

	s.logStream = os.Stdin
	os.Stdin = tty
}
