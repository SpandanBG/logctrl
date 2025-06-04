package main

import (
	"github.com/SpandanBG/logctrl/ui"
	"github.com/SpandanBG/logctrl/utils"
)

func main() {
	// create new app
	app := ui.New()

	// Setup user input signals
	utils.SetupCleanUpSignal(func() {
		app.Close()
	}, nil)

	// run app
	app.Run()
}
