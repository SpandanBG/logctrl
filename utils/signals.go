package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// OnTerminalResize - calls `callback` each time the terminal is resized (SIGWINCH).
func OnTerminalResize(callback func()) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGWINCH)

	go func() {
		for range sigCh {
			callback()
		}
	}()
}
