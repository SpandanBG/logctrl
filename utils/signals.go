package utils

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupCleanUpSignal - sets up signal for <Ctrl_c> pressed by user and calling
// cleanup job.
func SetupCleanUpSignal(cleanUp func(), proc *os.Process) {
	ctrl_c := make(chan os.Signal, 2)

	// Listen to <Ctrl_c> into channel
	signal.Notify(ctrl_c, os.Interrupt, syscall.SIGTERM)

	go func() {
		// wait for <Ctrl_c> user input
		<-ctrl_c

		// if any processes are running - terminate them
		if proc != nil {
			_ = proc.Kill()
		}

		// Call cleanUp for app clean up routine
		if cleanUp != nil {
			cleanUp()
		}

		// Graceful exit
		os.Exit(0)
	}()
}

func SetupEnterSignal(cleanUp func(), proc *os.Process) {
	enter := make(chan os.Signal, 2)

	// Listen to <Enter> into channel
	signal.Notify(enter, syscall.SIGPIPE)

	go func() {
		// wait for <Enter> user input
		for range enter {

			// Call cleanUp for app clean up routine
			if cleanUp != nil {
				cleanUp()
			}
		}

	}()
}
