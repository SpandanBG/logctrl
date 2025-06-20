package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui"
	"github.com/creack/pty"
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

	rP, wP, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe to child - %v", err)
	}

	console, err := os.Open("/dev/tty")
	if err != nil {
		log.Fatalf("unable to to open /dev/tty - %v", err)
	}

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
	}()

	go io.Copy(f, console)
	io.Copy(os.Stdout, f)
}

func runChild(childFd int) {
	logPipe := os.NewFile(uintptr(childFd), "logPipe")
	done := make(chan bool)

	go func() {
		bs := bufio.NewScanner(logPipe)
		for bs.Scan() {
			fmt.Println(bs.Text())
		}
		close(done)
	}()

	var input string
	fmt.Scan(&input)

	fmt.Println("you said:", input)

	<-done
}

func app() {
	// Attach log input source
	src := reader.ResolveSource(true)

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
