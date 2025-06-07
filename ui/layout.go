package ui

import (
	"log"

	"github.com/SpandanBG/logctrl/reader"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	appTitle   = "<Ctrl+LOG>"
	inputLabel = ":"
)

type App interface {
	Run()
	Close()
}

type app struct {
	tui    *tview.Application
	src    reader.Source
	root   *tview.Flex
	logBox *tview.TextView
	prompt *tview.InputField
}

// New - creates a new logctr app using tview package.
func New(src reader.Source) App {
	ttyScreen, err := tcell.NewTerminfoScreen()
	if err != nil {
		log.Fatalf("unable to create ttyp screen: %v", err.Error())
	}

	app := &app{
		tui: tview.NewApplication().SetScreen(ttyScreen),
		src: src,
	}

	app.createRoot()
	app.createLogBox()
	app.createPrompt()

	return app
}

// Run - starts the app. panics with an error if unable to run
func (a *app) Run() {
	// Attach stdin to log dialog
	go a.attachLog()

	// Registers default key strokes
	a.registerKeys()

	// Start app
	if err := a.tui.Run(); err != nil {
		log.Fatalf("app runtime err: %v", err.Error())
	}
}

// Close - ends the app.
func (a *app) Close() {
	a.src.Close()
	a.tui.Stop()
}

// --------------------- private methods and function

// createRoot - creates a fullscreen flex window as the root of the viewport.
func (a *app) createRoot() {
	a.root = tview.NewFlex().SetDirection(tview.FlexRow)
	a.tui.SetRoot(a.root, true).EnableMouse(true)
}

// createLogBox - creates a text view as log dialog box.
func (a *app) createLogBox() {
	a.logBox = tview.NewTextView().SetDynamicColors(true)
	a.root.AddItem(a.logBox, 0, 1, false)
}

// createPrompt - creates an input field with focused for user inputs.
func (a *app) createPrompt() {
	a.prompt = tview.NewInputField().SetLabel(inputLabel)
	a.root.AddItem(a.prompt, 1, 0, true)
}

// writeLog - writes the log string to the `logBox`
func (a *app) writeLog(log string) {
	a.tui.QueueUpdateDraw(func() {
		a.logBox.Write([]byte(log))
	})
}

// attachLog - attaches the stdin buffer from `src` to `logBox`
func (a *app) attachLog() {
	for log := range a.src.Stream() {
		a.writeLog(log + "\n")
	}

	a.writeLog("Log ended. Please press Ctrl+C to close.")

	a.tui.QueueUpdateDraw(func() {
		a.tui.SetFocus(a.prompt)
	})
}

// registerKeys - registrers all app realated keys strokes:
// - <Ctrl+c> : closes app
func (a *app) registerKeys() {
	a.tui.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			a.Close()
			return nil
		}

		return event
	})
}
