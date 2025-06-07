package main

import (
	"github.com/SpandanBG/logctrl/reader"
	"github.com/SpandanBG/logctrl/ui"
)

func main() {
	// Attach log input source
	src := reader.ResolveSource()

	// create new app
	app := ui.New(src)

	// run app
	app.Run()
}
