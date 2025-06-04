package ui

import "github.com/rivo/tview"

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
	root   *tview.Flex
	logBox *tview.TextView
	prompt *tview.InputField
}

// New - creates a new logctr app using tview package.
func New() App {
	app := &app{tui: tview.NewApplication()}

	app.createRoot()
	app.createLogBox()
	app.createPrompt()

	return app
}

// Run - starts the app. panics with an error if unable to run
func (a *app) Run() {
	if err := a.tui.Run(); err != nil {
		panic(err)
	}
}

// Close - ends the app.
func (a *app) Close() {
	// TODO: add to kill if any addtional process are added
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
