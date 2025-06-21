package ui

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	lines   []string
	logFile *os.File
	scanner *bufio.Scanner
}

type logMsg string

func NewUI(logFeed *os.File) (
	app *tea.Program,
	cleanup func(),
) {
	logFile, err := os.CreateTemp("", "logs.txt")
	if err != nil {
		log.Fatalf("unable to create temp log file - %v", err)
	}

	logReader, logWriter, err := os.Pipe()
	if err != nil {
		log.Fatalf("unable to create log reader - writer pipe - %v", err)
	}

	// start reading and writing to pipe and file
	go func() {
		teeReader := io.TeeReader(logFeed, logFile)
		io.Copy(logWriter, teeReader)
	}()

	app = tea.NewProgram(model{
		logFile: logFile,
		scanner: bufio.NewScanner(logReader),
		lines:   make([]string, 0),
	}, tea.WithAltScreen())

	cleanup = func() {
		defer logFile.Close()
		defer logReader.Close()
	}

	return app, cleanup
}

func (m model) Init() tea.Cmd {
	return m.nextLog()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// is it a key press?
	case tea.KeyMsg:
		return m.executeKey(msg.String())
	case logMsg:
		return m.updateLogLines(string(msg))
	}

	return m, nil
}

func (m model) View() string {
	return strings.Join(m.lines, "\r\n")
}

// -------------------- Private Methods

func (m model) executeKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	// should exit the program
	case "ctrl+c", "q":
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m model) updateLogLines(line string) (tea.Model, tea.Cmd) {
	m.lines = append(m.lines, line)
	return m, m.nextLog()
}

func (m model) nextLog() tea.Cmd {
	return func() tea.Msg {
		if m.scanner.Scan() {
			return logMsg(m.scanner.Text())
		}

		return logMsg("")
	}
}
