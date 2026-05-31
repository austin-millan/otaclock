// Package terminal implements the headless MONORITH DATAFLOW codec terminal.
// It mirrors the SDL boot log until the first submission, then switches to the
// shared session transcript.
package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

const (
	ansiReset       = "\u001B[0m"
	ansiGreen       = "\u001B[0;32m"
	ansiBright      = "\u001B[1;32m"
	ansiDim         = "\u001B[2;32m"
	ansiClearScreen = "\u001B[2J\u001B[H"
	ansiHideCursor  = "\u001B[?25l"
	ansiShowCursor  = "\u001B[?25h"

	termWidth         = 78
	innerWidth        = termWidth - 4
	bootViewportLines = 18
	footer            = "U.S MARINES DATAFLOW SYSTEM : CONFIDENTIAL"
)

// App is the headless terminal front-end.
type App struct {
	session *core.Session
	start   time.Time
}

// New creates a terminal app bound to a fresh session from the environment.
func New() *App {
	return &App{session: core.NewSessionFromEnvironment(), start: time.Now()}
}

// Run blocks, driving the boot animation and the interactive prompt loop.
func (a *App) Run() {
	a.installInterruptHandler()
	fmt.Print(ansiHideCursor)
	a.session.StartBootIfNeeded(a.nowMs())
	a.playBootAnimation()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		a.renderScreen()
		a.printPrompt()
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "/quit", "/exit":
			a.disconnect()
			return
		case "/clear":
			a.session.ClearTranscript()
			continue
		default:
			a.session.Submit(input, a.nowMs())
			a.waitWhileLoading()
			if a.session.Snapshot().ShowsBootScreen {
				a.playBootAnimation()
			}
		}
	}
	a.disconnect()
}

func (a *App) disconnect() {
	fmt.Printf("\n%s[ CODEC DISCONNECTED ]%s\n\n", ansiDim, ansiReset)
	fmt.Print(ansiShowCursor)
}

func (a *App) playBootAnimation() {
	for a.session.Snapshot().ShowsBootScreen && !a.session.IsBootComplete(a.nowMs()) {
		a.renderScreen()
		time.Sleep(32 * time.Millisecond)
	}
	a.renderScreen()
}

func (a *App) waitWhileLoading() {
	for a.session.Snapshot().IsLoading {
		a.session.Tick(a.nowMs())
		a.renderScreen()
		time.Sleep(100 * time.Millisecond)
	}
}

func (a *App) nowMs() uint32 {
	return uint32(time.Since(a.start).Milliseconds())
}

func (a *App) renderScreen() {
	snap := a.session.Snapshot()
	border := strings.Repeat("═", termWidth)

	fmt.Print(ansiClearScreen)
	fmt.Printf("%s╔%s╗%s\n", ansiGreen, border, ansiReset)

	if snap.ShowsBootScreen {
		a.renderBootBody()
	} else {
		a.renderTranscriptBody(snap)
	}

	fmt.Printf("%s╠%s╣%s\n", ansiGreen, border, ansiReset)
	a.printRow(ansiDim + footer + ansiReset)
	fmt.Printf("%s╚%s╝%s\n", ansiGreen, border, ansiReset)
}

func (a *App) renderBootBody() {
	frame := a.session.BootFrame(a.nowMs())
	visible := bootLinesForViewport(frame.Lines, bootViewportLines)

	a.printRow("")
	for index, line := range visible {
		isLast := index == len(visible)-1
		style := ansiDim
		cursor := ""
		if isLast {
			style = ansiBright
			if frame.ShowBlockCursor {
				cursor = "█"
			}
		}
		a.printRow(fmt.Sprintf("%s  %s%s%s", style, line, cursor, ansiReset))
	}
	for i := len(visible); i < bootViewportLines; i++ {
		a.printRow("")
	}
}

func (a *App) renderTranscriptBody(snap core.Snapshot) {
	if len(snap.Transcript) == 0 {
		a.printRow("")
		a.printRow(fmt.Sprintf("%s  [ awaiting input · help · login ]%s", ansiDim, ansiReset))
		a.printRow("")
		return
	}

	for _, msg := range snap.Transcript {
		prefix := label(msg.Role)
		style := ansiBright
		if msg.Role == core.RoleSystem {
			style = ansiDim
		}
		a.printWrapped(prefix, msg.Content, style)
		a.printRow("")
	}

	if snap.IsLoading {
		a.printRow(fmt.Sprintf("%s  Transmitting…%s", ansiDim, ansiReset))
	}
}

func bootLinesForViewport(lines []string, maxLines int) []string {
	if len(lines) <= maxLines {
		return lines
	}
	return lines[len(lines)-maxLines:]
}

func label(role core.Role) string {
	switch role {
	case core.RoleUser:
		return ">"
	case core.RoleAssistant:
		return "ota"
	default:
		return "sys"
	}
}

func (a *App) printRow(content string) {
	stripped := stripANSI(content)
	pad := termWidth - len([]rune(stripped))
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("%s║%s%s%s%s║%s\n", ansiGreen, ansiReset, content, strings.Repeat(" ", pad), ansiGreen, ansiReset)
}

func (a *App) printWrapped(prefix, text, style string) {
	maxLine := innerWidth - 6
	for _, paragraph := range strings.Split(text, "\n") {
		line := ""
		isFirst := true
		for _, word := range strings.Fields(paragraph) {
			switch {
			case line == "":
				line = word
			case len([]rune(line))+1+len([]rune(word)) <= maxLine:
				line += " " + word
			default:
				a.emitWrapped(prefix, line, &isFirst, style)
				line = word
			}
		}
		a.emitWrapped(prefix, line, &isFirst, style)
	}
}

func (a *App) emitWrapped(prefix, line string, isFirst *bool, style string) {
	if line == "" {
		return
	}
	label := "     "
	if *isFirst {
		label = fmt.Sprintf("  %s ", prefix)
	}
	a.printRow(fmt.Sprintf("%s%s%s%s", style, label, line, ansiReset))
	*isFirst = false
}

func (a *App) printPrompt() {
	fmt.Printf("\n%s>%s ", ansiDim, ansiReset)
}

func stripANSI(text string) string {
	var b strings.Builder
	skip := false
	for _, ch := range text {
		if ch == '\u001B' {
			skip = true
			continue
		}
		if skip {
			if ch == 'm' {
				skip = false
			}
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func (a *App) installInterruptHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGINT)
	go func() {
		<-ch
		fmt.Printf("\n%s[ CODEC DISCONNECTED ]%s\n\n", ansiDim, ansiReset)
		fmt.Print(ansiShowCursor)
		os.Exit(0)
	}()
}
