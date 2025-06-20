package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui"
	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
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
	childFdStr := os.Getenv("child")
	if len(childFdStr) > 0 {
		// Parse the fd number and drop into the child mode
		childFd, err := strconv.Atoi(childFdStr)
		if err != nil {
			log.Fatalf("unable to parse child env var - %v", err)
		}

		runChild(childFd) // never returns
		os.Exit(0)
	}

	// ──────────────────────────────
	// 2. Parent cleanup guarantee
	//    Ensure the upstream producer (A) is killed
	//    when *we* disappear.
	// ──────────────────────────────
	pg := unix.Getpgrp() // pipeline process-group id
	defer unix.Kill(pg, syscall.SIGTERM)

	// ──────────────────────────────
	// 3. Anonymous pipe for   A → B
	// ──────────────────────────────
	rP, wP, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe to child - %v", err)
	}

	// ──────────────────────────────
	// 4. Grab the real keyboard and
	//    put it in RAW so ^C etc.
	//    arrive as bytes.
	// ──────────────────────────────
	console, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("unable to to open /dev/tty - %v", err)
	}
	oldState, err := term.MakeRaw(int(console.Fd()))
	if err != nil {
		log.Fatalf("unable to turn console raw - %v", err)
	}
	defer term.Restore(int(console.Fd()), oldState)

	// ──────────────────────────────
	// 5. Re-exec ourselves as the
	//    child wrapped in a PTY.
	//    • CHILD   env advertises fd
	//    • ExtraFiles[0] becomes fd 3
	// ──────────────────────────────
	self := os.Args[0]
	cmd := exec.Command(self)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("child=%d", rP.Fd()))
	cmd.ExtraFiles = append(cmd.ExtraFiles, rP)

	// Spawn in a fresh PTY;  f = PTY master
	f, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("unable to start pty - %v", err)
	}

	// ──────────────────────────────
	// 6. Pump A’s output (parent’s
	//    stdin) into the pipe → child
	// ──────────────────────────────
	go func() {
		io.Copy(wP, os.Stdin)
		wP.Close()

		// When A finishes the kernel resets
		// tty to cooked. Force RAW again so
		// keystrokes keep flowing.
		if _, err := term.MakeRaw(int(console.Fd())); err != nil {
			log.Fatalf("unable to turn console raw - %v", err)
		}
	}()

	// ──────────────────────────────
	// 7. Pump keyboard → PTY master
	// ──────────────────────────────
	go io.Copy(f, console)

	// ──────────────────────────────
	// 8. Pump child screen → stdout
	// ──────────────────────────────
	io.Copy(os.Stdout, f)

	// ──────────────────────────────
	// 9. Close read-end in parent so
	//    we don’t infect EOF logic
	// ──────────────────────────────
	rP.Close()
}

// ────────────────────────────────────────────────────────────────────────────────
// Child-side TUI: displays log lines   +   takes user commands
// ────────────────────────────────────────────────────────────────────────────────
func runChild(childFd int) {
	// fd passed by parent becomes os.File
	logPipe := os.NewFile(uintptr(childFd), "logPipe")

	// tview widgets
	app := tview.NewApplication()
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	logBox := tview.NewTextView().SetDynamicColors(true)
	prompt := tview.NewInputField()

	root.AddItem(logBox, 0, 1, false) // growing log window
	root.AddItem(prompt, 1, 0, true)  // single-line prompt

	app.SetRoot(root, true).EnableMouse(true)

	// Set input captures of app
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		// Clear prompt on Enter
		case tcell.KeyEnter:
			prompt.SetText("")
			return nil
		}

		return event
	})

	// Background goroutine: read log lines from the pipe and append to logBox
	go func() {
		bs := bufio.NewScanner(logPipe)
		for bs.Scan() {
			app.QueueUpdateDraw(func() {
				logBox.Write(bs.Bytes())
			})
		}
	}()

	app.Run() // blocking
}

func app() {
	// Attach log input source
	src := reader.ResolveSource(true)

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
