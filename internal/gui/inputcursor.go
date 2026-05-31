package gui

import text "github.com/hajimehoshi/ebiten/v2/text/v2"

// inputCursorPos returns the wrapped line index and x pixel offset for a rune
// cursor index within text.
func inputCursorPos(text string, cursorRune int, maxWidth int, face text.Face) (lineIdx, x int) {
	runes := []rune(text)
	if cursorRune < 0 {
		cursorRune = 0
	}
	if cursorRune > len(runes) {
		cursorRune = len(runes)
	}
	if cursorRune == 0 {
		return 0, 0
	}
	wrapped := wrapText(string(runes[:cursorRune]), maxWidth, face)
	if len(wrapped) == 0 {
		return 0, 0
	}
	return len(wrapped) - 1, textWidth(wrapped[len(wrapped)-1], face)
}

func clampInputCursor(text string, cursor int) int {
	runes := []rune(text)
	if cursor < 0 {
		return 0
	}
	if cursor > len(runes) {
		return len(runes)
	}
	return cursor
}
