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
	"github.com/SpandanBG/logctrl/utils"
	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	if os.Getenv("child") != "" {
		pipeFD, _ := strconv.Atoi(os.Getenv("child"))
		pipe := os.NewFile(uintptr(pipeFD), "log")

		app := tview.NewApplication()
		root := tview.NewFlex().SetDirection(tview.FlexRow)

		logView := tview.NewTextView()
		root.AddItem(logView, 0, 1, false)

		prompt := tview.NewInputField().SetLabel(":")
		root.AddItem(prompt, 1, 0, true)

		app.SetRoot(root, true).EnableMouse(true)

		app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyCtrlC:
				app.Stop()
				return nil
			case tcell.KeyEnter:
				prompt.SetText("")
				return nil
			}

			return event
		})

		go app.QueueUpdateDraw(func() {
			app.SetFocus(prompt)
		})

		go func() {
			in := bufio.NewScanner(pipe)
			for in.Scan() {
				app.QueueUpdateDraw(func() {
					logView.Write([]byte(in.Text() + "\n"))
				})
			}
		}()

		app.Run()
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
		pw.Close()
	}()

	tty, _ := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	go io.Copy(f, tty)

	utils.SetupCleanUpSignal(func() {
		f.Write([]byte{0x03})
		_ = child.Wait()
	}, nil)

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
