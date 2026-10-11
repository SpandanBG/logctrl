package utils

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func Test_OnTerminalResizeRunsCallbackOnEachResize(t *testing.T) {
	resized := make(chan struct{}, 2)
	OnTerminalResize(func() { resized <- struct{}{} })

	for range 2 {
		if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
			t.Fatalf("unable to send SIGWINCH - %v", err)
		}

		select {
		case <-resized:
		case <-time.After(time.Second):
			t.Fatal("callback not run after terminal resize")
		}
	}
}
