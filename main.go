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

func main() {
	childFdStr := os.Getenv("child")
	if len(childFdStr) > 0 {
		childFd, err := strconv.Atoi(childFdStr)
		if err != nil {
			log.Fatalf("unable to parse child env var - %v", err)
		}

		runChild(childFd)
		os.Exit(0)
	}

	pg := unix.Getpgrp()
	defer unix.Kill(pg, syscall.SIGTERM)

	rP, wP, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe to child - %v", err)
	}

	console, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("unable to to open /dev/tty - %v", err)
	}

	oldConsole, err := term.MakeRaw(int(console.Fd()))
	if err != nil {
		log.Fatalf("unable to turn console raw - %v", err)
	}
	defer term.Restore(int(console.Fd()), oldConsole)

	self := os.Args[0]
	cmd := exec.Command(self)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("child=%d", rP.Fd()))
	cmd.ExtraFiles = append(cmd.ExtraFiles, rP)

	f, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("unable to start pty - %v", err)
	}

	go func() {
		io.Copy(wP, os.Stdin)
		wP.Close()

		if _, err := term.MakeRaw(int(console.Fd())); err != nil {
			log.Fatalf("unable to turn console raw - %v", err)
		}
	}()

	go io.Copy(f, console)
	io.Copy(os.Stdout, f)
}

func runChild(childFd int) {
	logPipe := os.NewFile(uintptr(childFd), "logPipe")
	done := make(chan bool)

	app := tview.NewApplication()
	root := tview.NewFlex().SetDirection(tview.FlexRow)

	logBox := tview.NewTextView().SetDynamicColors(true)
	prompt := tview.NewInputField()

	root.AddItem(logBox, 0, 1, false)
	root.AddItem(prompt, 1, 0, true)

	app.SetRoot(root, true).EnableMouse(true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			prompt.SetText("")
			return nil
		}

		return event
	})

	go func() {
		bs := bufio.NewScanner(logPipe)
		for bs.Scan() {
			app.QueueUpdateDraw(func() {
				logBox.Write(bs.Bytes())
			})
		}
		close(done)
	}()

	app.Run()
}

func app() {
	// Attach log input source
	src := reader.ResolveSource(true)

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
