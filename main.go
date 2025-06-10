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
	if os.Getenv("child") != "" {
		ch := make(chan bool)

		go func() {
			pipeFd, err := strconv.Atoi(os.Getenv("child"))
			if err != nil {
				log.Fatalf("unable to parse to int of child env: %v", err)
			}

			logPipe := os.NewFile(uintptr(pipeFd), "logpipe")

			log := bufio.NewScanner(logPipe)
			for log.Scan() {
				fmt.Println(log.Text())
			}

			close(ch)
		}()

		<-ch

		return
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create pipe: %v", err)
	}

	child := exec.Command(os.Args[0])
	child.Env = append(
		child.Environ(),
		fmt.Sprintf("child=%d", pr.Fd()),
	)
	child.ExtraFiles = append(child.ExtraFiles, pr)

	f, _ := pty.Start(child)

	go func() {
		io.Copy(pw, os.Stdin)
		pw.Write([]byte("bye"))
		pw.Close()
	}()

	io.Copy(os.Stdout, f)
}

func app() {
	// Attach log input source
	src := reader.ResolveSource(true)

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
