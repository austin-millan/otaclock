package gui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// termLine is one styled line of the DATAFLOW terminal.
type termLine struct {
	text  string
	color color.Color
}

const terminalFooterText = "U.S MARINES DATAFLOW SYSTEM : CONFIDENTIAL"

// terminalFrameLayout holds pixel geometry for the MGX DATAFLOW frame: an open
// rectangle with top and bottom horizontals that overhang left of the vertical,
// no right border, and a small accent block outside the lower-left corner.
type terminalFrameLayout struct {
	frameLeft     int
	frameRight    int
	lineLeft      int
	innerTop      int
	frameBottom   int
	contentTop    int
	contentBottom int
	contentH      int
	textX         int
	textWidth     int
}

func terminalFrameLayoutFor(x, y, width, height int, m LayoutMetrics) terminalFrameLayout {
	inset := m.Scaled(10)
	footerBand := m.Scaled(22)
	textPadX := m.Scaled(10)
	textPadY := m.Scaled(8)
	overhang := m.Scaled(8)

	frameLeft := x + inset
	frameRight := x + width - inset
	lineLeft := frameLeft - overhang
	innerTop := y + inset
	frameBottom := y + height - footerBand
	contentTop := innerTop + textPadY
	contentBottom := frameBottom - m.Scaled(4)
	contentH := maxInt(0, contentBottom-contentTop)
	textX := frameLeft + textPadX

	return terminalFrameLayout{
		frameLeft:     frameLeft,
		frameRight:    frameRight,
		lineLeft:      lineLeft,
		innerTop:      innerTop,
		frameBottom:   frameBottom,
		contentTop:    contentTop,
		contentBottom: contentBottom,
		contentH:      contentH,
		textX:         textX,
		textWidth:     maxInt(0, frameRight-textX-textPadX),
	}
}

// drawTerminalFrame draws the MGX DATAFLOW border: top and bottom lines extend
// past the left vertical; there is no right vertical; the accent block sits
// outside the frame to the left of the vertical near the bottom.
func drawTerminalFrame(dst *ebiten.Image, l terminalFrameLayout, m LayoutMetrics) {
	border := Theme.TerminalBorder
	horizSpan := l.frameRight - l.lineLeft + 1

	fillRect(dst, l.lineLeft, l.innerTop, horizSpan, 1, border)
	fillRect(dst, l.lineLeft, l.frameBottom, horizSpan, 1, border)
	fillRect(dst, l.frameLeft, l.innerTop, 1, l.frameBottom-l.innerTop+1, border)

	blockW := m.Scaled(6)
	blockH := m.Scaled(4)
	gap := m.Scaled(2)
	fillRect(dst, l.frameLeft-blockW-gap, l.frameBottom-blockH+1, blockW, blockH, border)
}

// renderTerminalContent draws the framed DATAFLOW terminal. scrollOffset is the
// number of lines scrolled up from the bottom (0 == newest).
func renderTerminalContent(
	dst *ebiten.Image,
	x, y, width, height int,
	lines []termLine,
	m LayoutMetrics,
	scrollOffset int,
	showBlockCursor bool,
	face text.Face,
) {
	fillRect(dst, x, y, width, height, Theme.TerminalBackground)

	l := terminalFrameLayoutFor(x, y, width, height, m)
	drawTerminalFrame(dst, l, m)

	if face == nil || l.contentH <= 0 {
		drawTerminalFooter(dst, l.frameLeft, l.frameBottom, m, face)
		return
	}

	lineHeight := maxInt(faceHeight(face)-1, m.Scaled(10))
	maxVisible := maxInt(1, l.contentH/lineHeight)
	endIdx := len(lines) - scrollOffset
	startIdx := maxInt(0, endIdx-maxVisible)
	upper := minInt(endIdx, len(lines))

	textY := l.contentTop
	lastGlobalIdx := len(lines) - 1
	for globalIdx := startIdx; globalIdx < upper; globalIdx++ {
		line := lines[globalIdx]
		drawText(dst, line.text, l.textX, textY, face, line.color)

		if globalIdx == lastGlobalIdx && showBlockCursor {
			tw := textWidth(line.text, face)
			th := faceHeight(face)
			fillRect(dst,
				l.textX+tw+m.Scaled(4), textY,
				m.Scaled(9), maxInt(lineHeight-2, th),
				line.color)
		}

		textY += lineHeight
		if textY >= l.contentBottom {
			break
		}
	}

	drawTerminalFooter(dst, l.frameLeft, l.frameBottom, m, face)
}

// renderTerminalBoot draws the boot sequence (plain strings, pinned to bottom).
func renderTerminalBoot(
	dst *ebiten.Image,
	x, y, width, height int,
	lines []string,
	m LayoutMetrics,
	showBlockCursor bool,
	face text.Face,
) {
	colored := make([]termLine, len(lines))
	for index, t := range lines {
		colored[index] = termLine{text: t, color: Theme.TerminalForeground}
	}
	renderTerminalContent(dst, x, y, width, height, colored, m, 0, showBlockCursor, face)
}

func renderInputChrome(dst *ebiten.Image, x, y, width, height int) {
	fillRect(dst, x, y, width, height, Theme.TerminalBackground)
	fillRect(dst, x, y, width, 1, Theme.TerminalBorder)
}

func drawTerminalFooter(dst *ebiten.Image, frameLeft, frameBottom int, m LayoutMetrics, face text.Face) {
	drawText(dst, terminalFooterText, frameLeft, frameBottom+m.Scaled(6), face, Theme.TerminalForeground)
}
