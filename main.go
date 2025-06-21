package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"

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
	// ──────────────────────────────
	// 1. Are we the re-exec’ed child?
	// ──────────────────────────────
	if len(os.Getenv(ChildEnvVar)) > 0 {
		startChildProcess()
		os.Exit(0)
	}

	// ──────────────────────────────
	// 2. Parent cleanup guarantee
	//    Ensure the upstream producer (A) is killed
	//    when *we* disappear.
	// ──────────────────────────────
	pgid := unix.Getpgrp()
	defer unix.Kill(pgid, syscall.SIGTERM)

	// ──────────────────────────────
	// 3. Anonymous pipe for   A → B.
	// 	  Grab the real keyboard and
	//    put it in RAW so ^C etc.
	//    arrive as bytes.
	// ──────────────────────────────
	logReader, logWritter, console, cleanup := setupStreaming()
	defer cleanup()

	// ──────────────────────────────
	// 4. Re-exec ourselves as the
	//    child wrapped in a PTY.
	//    • CHILD   env advertises fd
	//    • ExtraFiles[0] becomes fd 3
	// ──────────────────────────────
	ptm := startPTY(logReader)

	// ──────────────────────────────
	// 6. Pump A’s output (parent’s
	//    stdin) into the pipe → child
	// ──────────────────────────────
	startDataPump(ptm, logWritter, console)
}

func startChildProcess() {
	// Parse the fd number and drop into the child mode
	childFdStr := os.Getenv(ChildEnvVar)
	childFd, err := strconv.Atoi(childFdStr)
	if err != nil {
		log.Fatalf("unable to parse child env var - %v", err)
	}

	logPipe := os.NewFile(uintptr(childFd), "logPipe")

	io.Copy(os.Stdout, logPipe)
}

func setupStreaming() (
	logReader,
	logWritter,
	console *os.File,
	cleanup func(),
) {
	var err error

	logReader, logWritter, err = os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe to child - %v", err)
	}

	console, err = os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("unable to to open /dev/tty - %v", err)
	}

	oldState, err := term.MakeRaw(int(console.Fd()))
	if err != nil {
		log.Fatalf("unable to turn console raw - %v", err)
	}

	cleanup = func() {
		defer term.Restore(int(console.Fd()), oldState)
	}

	return
}

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

	return ptm
}

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
