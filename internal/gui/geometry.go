package gui

import "github.com/hajimehoshi/ebiten/v2"

// spriteScreenLeft returns the sprite's current on-screen left edge, derived from
// the live window position and the current uiMode/appW.
func (g *Game) spriteScreenLeft() int {
	winX, _ := ebiten.WindowPosition()
	return winX + g.spriteOriginX()
}

// anchorSpriteLeft repositions the window so the sprite's on-screen left edge
// matches spriteLeft under the current uiMode and appW. Callers must capture
// spriteLeft via spriteScreenLeft before mutating either, so the sprite stays
// visually fixed across width and mode changes.
func (g *Game) anchorSpriteLeft(spriteLeft int) {
	_, winY := ebiten.WindowPosition()
	ebiten.SetWindowPosition(spriteLeft-g.spriteOriginX(), winY)
}

func (g *Game) toggleChatPanel() {
	g.setChatPanel(g.uiMode == modeCompact)
}

func (g *Game) setChatPanel(visible bool) {
	if g.menuVisible {
		g.hideMenu()
	}
	spriteLeft := g.spriteScreenLeft()
	if visible {
		g.uiMode = modeChat
		g.scrollOffset = 0
		g.responseH = g.layout.DefaultResponseHeight()
		g.inputH = g.layout.InputHeight()
		g.session.StartBootIfNeeded(g.nowMs())
		g.appH = g.layout.ChatWindowHeight(g.responseH, g.inputH)
		g.responseH = maxInt(g.layout.MinResponseHeight(), g.appH-g.layout.ChatPanelY()-g.inputH)
		g.appW = g.layout.ChatWindowWidth()
		g.anchorSpriteLeft(spriteLeft)
		g.recomputeInputLayout()
	} else {
		g.inputText = ""
		g.inputScrollLines = 0
		g.inputH = g.layout.InputHeight()
		g.uiMode = modeCompact
		g.appW = g.layout.WindowWidth()
		g.anchorSpriteLeft(spriteLeft)
		g.appH = g.layout.CompactWindowHeight()
	}
}

func (g *Game) applyChatWindowHeight(h int) {
	targetH := g.layout.ClampChatHeight(h)
	g.responseH = maxInt(g.layout.MinResponseHeight(), targetH-g.layout.ChatPanelY()-g.inputH)
	g.appH = targetH
}

func (g *Game) recomputeInputLayout() {
	if g.uiMode != modeChat || g.fonts == nil {
		return
	}
	face := g.fonts.Normal
	textX := g.layout.Pad() + g.layout.InputTextIndent()
	textW := g.appW - textX - g.layout.Pad()
	lines := wrapText(g.inputText, textW, face)

	needed := maxInt(1, len(lines))*g.lineHeight() + g.layout.Scaled(8)
	desired := minInt(g.layout.MaxInputHeight(), maxInt(g.layout.InputHeight(), needed))

	if desired != g.inputH {
		delta := desired - g.inputH
		g.inputH = desired
		g.applyChatWindowHeight(g.appH + delta)
	}

	visibleLines := maxInt(1, (g.inputH-g.layout.Scaled(8))/g.lineHeight())
	g.inputScrollLines = minInt(g.inputScrollLines, maxInt(0, len(lines)-visibleLines))
}
