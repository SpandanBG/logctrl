package main

import (
	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui"
	"github.com/SpandanBG/logctrl/utils"
)

func main() {
	// Attach log input source
	src := reader.ResolveSource()

	// create new app
	app := ui.New(src)

	// Setup user input signals
	utils.SetupCleanUpSignal(func() {
		app.Close()
	}, nil)

	// run app
	app.Run()
}
