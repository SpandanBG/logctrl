package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui"
	"github.com/SpandanBG/logctrl/utils"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	ChildEnvVar = "logctrl_child"
)

// ────────────────────────────────────────────────────────────────────────────────
// High-level flow
// ────────────────────────────────────────────────────────────────────────────────
// 1. First run (the “parent”):
//   - Creates an anonymous pipe        →  carries log lines from A to B.
//   - Opens /dev/tty in raw mode       →  gets real-time keystrokes.
//   - Re-execs itself as the “child”,  →  passing the read-end of the pipe in
//     wrapped in a PTY.                   ExtraFiles and advertising its fd
//     number via the CHILD env var.
//   - Pumps three directions:
//     A → pipe → child (log stream)
//     /dev/tty → PTY master  (user input)
//     PTY master → stdout     (child’s screen output)
//   - When it exits it sends SIGTERM to the whole pipeline PG.
//
// 2. Second run (the “child”):
//   - Builds a small TUI (tview) that reads the pipe (fd passed by parent)
//     and still accepts interactive input through the PTY slave.
//
// ────────────────────────────────────────────────────────────────────────────────
func main() {
	// Are we the re-exec’ed child?
	if len(os.Getenv(ChildEnvVar)) > 0 {
		startChildProcess()
		os.Exit(0)
	}

	// Parent cleanup guarantee.
	// Ensure the upstream producer (A) is killed
	// when *we* disappear.
	pgid := unix.Getpgrp()
	defer unix.Kill(pgid, syscall.SIGTERM)

	// Anonymous pipe for   A → B.
	// Grab the real keyboard and
	// put it in RAW so ^C etc.
	// arrive as bytes.
	logReader, logWritter, console, cleanup := setupStreaming()
	defer cleanup()

	// Re-exec ourselves as the
	// child wrapped in a PTY.
	// • CHILD   env advertises fd
	// • ExtraFiles[0] becomes fd 3
	ptm := startPTY(logReader)

	// Pump A’s output (parent’s
	// stdin) into the pipe → child
	startDataPump(ptm, logWritter, console)
}

// startChildProcess - prepares the stream and launches the app UI.
func startChildProcess() {
	// Parse the fd number and drop into the child mode
	childFdStr := os.Getenv(ChildEnvVar)
	childFd, err := strconv.Atoi(childFdStr)
	if err != nil {
		log.Fatalf("unable to parse child env var - %v", err)
	}

	// Get the log feed pipe
	logFeed := os.NewFile(uintptr(childFd), "logFeed")

	stream := reader.NewStream(logFeed)
	app, exit := ui.NewUI(stream)
	defer exit()

	if _, err := app.Run(); err != nil {
		log.Fatalf("unable to run app - %v", err)
	}
}

// setupStreaming - creates a log feed pipe and prepares the `/dev/tty` as the
// console for I/O by PTY. Ensures to make `tty` as raw to pass all handing of
// keyboard signals by child instead of parent.
//
// ensure to call `defer cleanup()` upon receiving returned items.
func setupStreaming() (
	logReader,
	logWritter,
	console *os.File,
	cleanup func(),
) {
	var err error

	// Creates the reader writer pipe for stream coming in from os.Stdin
	logReader, logWritter, err = os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe to child - %v", err)
	}

	// Opens up `/dev/tty` for passing all keyboard inputs to PTY
	console, err = os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("unable to to open /dev/tty - %v", err)
	}

	// Marks the current TTY as raw to ignore all keyboard signals (e.g. CTRL+c)
	// and pass those to the PTY to be handled by the child.
	oldState, err := term.MakeRaw(int(console.Fd()))
	if err != nil {
		log.Fatalf("unable to turn console raw - %v", err)
	}

	// cleanup - on call ensures to restore the terminal to original state on exit.
	cleanup = func() {
		defer term.Restore(int(console.Fd()), oldState)
	}

	return
}

// startPTY - starts self in a PTY and passes the required log feed reader file
// and set required env variables.
func startPTY(logReader *os.File) (ptm *os.File) {
	self := os.Args[0]
	cmd := exec.Command(self)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("%s=%d", ChildEnvVar, logReader.Fd()))
	cmd.ExtraFiles = append(cmd.ExtraFiles, logReader)

	// Spawn in a fresh PTY;  ptm = PTY master
	ptm, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("unable to start pty - %v", err)
	}
	defer resizePty(ptm)

	// Setup window resize signal
	utils.OnTerminalResize(func() {
		resizePty(ptm)
	})

	return ptm
}

// startDataPump
//   - perform writing to `logWritter` from `os.Stdin`.
//   - pumps keyboard input from `console` to `ptm`.
//   - pumps pty output's  to `os.Stdout`.
func startDataPump(ptm, logWritter, console *os.File) {
	// Pump A’s output (parent’s stdin) into the pipe → child
	go func() {
		io.Copy(logWritter, os.Stdin)
		logWritter.Close()

		// When A finishes the kernel resets
		// tty to cooked. Force RAW again so
		// keystrokes keep flowing.
		if _, err := term.MakeRaw(int(console.Fd())); err != nil {
			log.Fatalf("unable to turn console raw - %v", err)
		}
	}()

	// Pump keyboard → PTY master
	go io.Copy(ptm, console)

	// Pump child screen → stdout
	io.Copy(os.Stdout, ptm)
}

func resizePty(ptm *os.File) {
	// Setup PTY window size to parent window size
	if err := pty.InheritSize(os.Stdout, ptm); err != nil {
		log.Fatalf("unable to resize pty to parent window size - %v", err)
	}
}
