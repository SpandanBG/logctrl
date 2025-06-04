package ui

import "github.com/rivo/tview"

const (
	appTitle   = "<Ctrl+LOG>"
	inputLabel = ":"
)

type App interface {
	Run()
}

type app struct {
	tui    *tview.Application
	root   *tview.Flex
	logBox *tview.Box
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

// --------------------- private methods and function

// createRoot - creates a fullscreen flex window as the root of the viewport.
func (a *app) createRoot() {
	a.root = tview.NewFlex().SetDirection(tview.FlexRow)
	a.tui.SetRoot(a.root, true).EnableMouse(true)
}

// createLogBox - creates a box with app title at the top as log dialog box.
func (a *app) createLogBox() {
	a.logBox = tview.NewBox().SetBorder(true).SetTitle(appTitle)
	a.root.AddItem(a.logBox, 0, 1, false)
}

// createPrompt - creates an input field with focused for user inputs.
func (a *app) createPrompt() {
	a.prompt = tview.NewInputField().SetLabel(inputLabel)
	a.root.AddItem(a.prompt, 1, 0, true)
}
