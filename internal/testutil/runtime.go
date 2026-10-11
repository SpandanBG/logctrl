package testutil

import (
	"reflect"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// settleWait - how long the runtime waits for a message before deciding the
// model has settled.
const settleWait = 100 * time.Millisecond

// windowSizeRequest - the message `tea.WindowSize()` asks the runtime with.
var windowSizeRequest = reflect.TypeOf(tea.WindowSize()())

// Runtime - runs tea commands in the background like the tea runtime, so tests
// can drive a model through its public Update/View. Like the tea runtime, it
// answers `tea.WindowSize()` with the size of its terminal.
type Runtime struct {
	msgs     chan tea.Msg
	terminal tea.WindowSizeMsg
}

// NewRuntime - a runtime for a terminal `width` columns by `height` rows.
func NewRuntime(width, height int) *Runtime {
	return &Runtime{
		msgs:     make(chan tea.Msg, 64),
		terminal: tea.WindowSizeMsg{Width: width, Height: height},
	}
}

// Run - runs `cmd` (and every command in a batch) in the background.
func (r *Runtime) Run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, each := range batch {
				r.Run(each)
			}
			return
		}
		if reflect.TypeOf(msg) == windowSizeRequest {
			msg = r.terminal
		}
		r.msgs <- msg
	}()
}

// Settle - updates `model` with every message the runtime produces, running
// the returned commands too, until no message arrives for a short while.
func (r *Runtime) Settle(model tea.Model) tea.Model {
	for {
		select {
		case msg := <-r.msgs:
			var cmd tea.Cmd
			model, cmd = model.Update(msg)
			r.Run(cmd)
		case <-time.After(settleWait):
			return model
		}
	}
}
