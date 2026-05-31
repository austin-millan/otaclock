package gui

import (
	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

// cursorBlinkMs is the half-period of the input cursor blink (on for one
// interval, off for the next).
const cursorBlinkMs = 500

func (g *Game) renderResponsePanel(screen *ebiten.Image, snap core.Snapshot) {
	panelY := g.layout.ChatPanelY() + 1
	var face text.Face
	if g.fonts != nil {
		face = g.fonts.Small
	}
	now := g.nowMs()

	if snap.ShowsBootScreen {
		bootFrame := g.session.BootFrame(now)
		// The only blinking cursor lives in the input box; the boot log shows none.
		renderTerminalBoot(screen, 0, panelY, g.appW, g.responseH, bootFrame.Lines, g.layout, false, face)
		return
	}

	allLines := g.buildTranscriptLines(snap, face)
	maxVisible := g.mgxResponseVisibleLineCount()
	g.scrollOffset = minInt(g.scrollOffset, maxInt(0, len(allLines)-maxVisible))

	renderTerminalContent(screen, 0, panelY, g.appW, g.responseH, allLines, g.layout, g.scrollOffset, false, face)
}

func (g *Game) renderInputBar(screen *ebiten.Image, snap core.Snapshot) {
	barY := g.layout.ChatPanelY() + g.responseH
	var inputFace text.Face
	if g.fonts != nil {
		inputFace = g.fonts.Normal
	}

	renderInputChrome(screen, 0, barY+1, g.appW, g.inputH)

	textX := g.layout.Pad() + g.layout.InputTextIndent()
	textY := barY + g.layout.InputTextPadY()
	textW := g.appW - textX - g.layout.Pad()

	lines := wrapText(g.inputText, textW, inputFace)
	visibleLines := maxInt(1, (g.inputH-g.layout.Scaled(8))/g.lineHeight())
	g.inputScrollLines = minInt(g.inputScrollLines, maxInt(0, len(lines)-visibleLines))

	end := len(lines) - g.inputScrollLines
	start := maxInt(0, end-visibleLines)

	y := textY
	for idx := start; idx < end; idx++ {
		line := lines[idx]
		if idx == len(lines)-1 && g.showInputCursor(snap) {
			line += "▌"
		}
		drawText(screen, line, textX, y, inputFace, Theme.TerminalForeground)
		y += g.lineHeight()
	}
}

// showInputCursor reports whether the single blinking input cursor should be
// drawn this frame. It is the only cursor in the widget: it stays hidden while
// the boot sequence is still playing and blinks once the prompt is ready.
func (g *Game) showInputCursor(snap core.Snapshot) bool {
	now := g.nowMs()
	bootInProgress := snap.ShowsBootScreen && !g.session.IsBootComplete(now)
	if bootInProgress {
		return false
	}
	return (now/cursorBlinkMs)%2 == 0
}

func (g *Game) mgxInnerTextWidth() int {
	return terminalFrameLayoutFor(0, g.layout.ChatPanelY()+1, g.appW, g.responseH, g.layout).textWidth
}

func (g *Game) mgxResponseVisibleLineCount() int {
	l := terminalFrameLayoutFor(0, g.layout.ChatPanelY()+1, g.appW, g.responseH, g.layout)
	var face text.Face
	if g.fonts != nil {
		face = g.fonts.Small
	}
	lineHeightPx := maxInt(faceHeight(face)-1, g.layout.Scaled(10))
	if lineHeightPx <= 0 {
		lineHeightPx = g.layout.Scaled(10)
	}
	return maxInt(1, l.contentH/lineHeightPx)
}

func (g *Game) buildTranscriptLines(snap core.Snapshot, face text.Face) []termLine {
	textW := g.mgxInnerTextWidth()
	var lines []termLine

	for _, msg := range snap.Transcript {
		switch msg.Role {
		case core.RoleSystem:
			for _, ln := range wrapText(msg.Content, textW, face) {
				lines = append(lines, termLine{text: ln, color: Theme.TerminalDim})
			}
		case core.RoleUser:
			wrapped := wrapText(msg.Content, textW-g.layout.Scaled(16), face)
			for idx, ln := range wrapped {
				prefixed := "  " + ln
				if idx == 0 {
					prefixed = "> " + ln
				}
				lines = append(lines, termLine{text: prefixed, color: Theme.TerminalForeground})
			}
		case core.RoleAssistant:
			for _, ln := range wrapText(msg.Content, textW, face) {
				lines = append(lines, termLine{text: ln, color: Theme.TerminalForeground})
			}
		}
		lines = append(lines, termLine{text: "", color: Theme.TerminalForeground})
	}

	if snap.IsLoading {
		dots := []string{".", "..", "..."}[(g.nowMs()/400)%3]
		lines = append(lines, termLine{text: "Transmitting" + dots, color: Theme.TerminalDim})
	}
	if len(lines) == 0 && !snap.ShowsBootScreen {
		lines = append(lines, termLine{text: "[ help · login · ESC ]", color: Theme.TerminalDim})
	}
	return lines
}
