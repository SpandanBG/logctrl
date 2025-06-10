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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	if os.Getenv("child") != "" {
		pipeFD, _ := strconv.Atoi(os.Getenv("child"))
		pipe := os.NewFile(uintptr(pipeFD), "log")

		groupFD, _ := strconv.Atoi(os.Getenv("group"))
		group := os.NewFile(uintptr(groupFD), "group")

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

		go func() {
		}()

		go app.QueueUpdateDraw(func() {
			app.SetFocus(prompt)
		})

		go io.Copy(os.Stdin, group)

		go func() {
			in := bufio.NewScanner(pipe)
			for in.Scan() {
				app.QueueUpdateDraw(func() {
					logView.Write([]byte(in.Text() + "\n"))
				})
			}
		}()

		if err := app.Run(); err != nil {
			log.Fatalf("error runing %v", err)
		}
		return
	}

	pr, pw, _ := os.Pipe()
	gr, gw, _ := os.Pipe()

	child := exec.Command(os.Args[0])

	child.Env = append(
		child.Environ(),
		fmt.Sprintf("child=%d", pr.Fd()),
		fmt.Sprintf("group=%d", gr.Fd()),
	)

	child.ExtraFiles = append(child.ExtraFiles, pr, gr)

	utils.SetupCleanUpSignal(func() {
		gw.Write([]byte{0x03})
		_ = child.Wait()
	}, nil)

	utils.SetupEnterSignal(func() {
		gw.Write([]byte{0xd})
	}, nil)

	go func() {
		io.Copy(pw, os.Stdin)
		pw.Close()
	}()

	child.Run()
	if err := child.Wait(); err != nil {
		log.Fatalf("didn't wait for the child: %v", err)
	}
}

func app() {
	// Attach log input source
	src := reader.ResolveSource(true)

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
