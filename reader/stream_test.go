package reader

import (
	"fmt"
	"os"
	"testing"

	"github.com/SpandanBG/logctrl/internal/testutil"
)

// startStream - starts a stream over a real pipe. Returns the write end and the
// notification channel; both are cleaned up when the test ends.
func startStream(t *testing.T, height int) (Stream, *os.File, chan bool) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	next := make(chan bool)
	s := NewStream(r)
	s.SetBufferSize(height)
	s.Start(next)

	t.Cleanup(func() {
		w.Close()
		s.Close()
	})
	return s, w, next
}

func Test_StreamWindowHasLineWhenNotified(t *testing.T) {
	s, w, next := startStream(t, 1)

	for i := range 200 {
		fmt.Fprintln(w, i)
		<-next
		testutil.Equal(t, fmt.Sprint(i), s.GetWindow())
	}
}

func Test_StreamPinnedWhileIngesting(t *testing.T) {
	s, w, next := startStream(t, 2)
	for i := range 10 {
		fmt.Fprintln(w, i)
		<-next
	}

	testutil.Equal(t, -3, s.MoveBy(-3))

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 10; i < 15; i += 1 {
			fmt.Fprintln(w, i)
			<-next
		}
	}()
	for {
		testutil.Equal(t, "5\n6", s.GetWindow())
		select {
		case <-done:
		default:
			continue
		}
		break
	}

	s.MoveToEnd()
	testutil.Equal(t, "13\n14", s.GetWindow())
}
