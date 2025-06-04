package ui

import "github.com/rivo/tview"

type App interface {
	Run()
}

type app struct {
	tui *tview.Application
}

// New - creates a new logctr app using tview package.
func New() App {
	tui := tview.NewApplication()

	return &app{
		tui: tui,
	}
}

// Run - starts the app. panics with an error if unable to run
func (a *app) Run() {
	if err := a.tui.Run(); err != nil {
		panic(err)
	}
}
