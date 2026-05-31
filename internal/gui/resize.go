package gui

import "github.com/hajimehoshi/ebiten/v2"

type resizeHandle int

const (
	handleNone resizeHandle = iota
	handleEast
	handleSouth
	handleSouthEast
	handleWest
	handleSouthWest
)

func (g *Game) globalMouse() (int, int) {
	cx, cy := ebiten.CursorPosition()
	wx, wy := ebiten.WindowPosition()
	return wx + cx, wy + cy
}

func (g *Game) resizeHandleAt(x, y int) resizeHandle {
	m := g.layout.ResizeMargin()
	onLeft := x < m
	onRight := x >= g.appW-m
	onBottom := y >= g.appH-m

	if g.uiMode == modeChat {
		if onLeft && onBottom {
			return handleSouthWest
		}
		if onLeft {
			return handleWest
		}
	}
	if onRight && onBottom {
		return handleSouthEast
	}
	if onRight {
		return handleEast
	}
	if onBottom {
		return handleSouth
	}
	return handleNone
}

func (g *Game) beginResize(handle resizeHandle) {
	if handle == handleNone {
		return
	}
	g.isResizing = true
	g.resizeHandle = handle
	g.resizeStartScale = g.widgetScale
	g.resizeStartW = g.appW
	g.resizeStartH = g.appH
	g.resizeStartMouseX, g.resizeStartMouseY = g.globalMouse()
	g.resizeStartWinX, g.resizeStartWinY = ebiten.WindowPosition()
}

func (g *Game) updateResizeDrag() {
	if !g.isResizing {
		return
	}
	absX, absY := g.globalMouse()
	dx := absX - g.resizeStartMouseX
	dy := absY - g.resizeStartMouseY

	newW := g.resizeStartW
	newH := g.resizeStartH

	switch g.resizeHandle {
	case handleEast, handleSouthEast:
		newW = maxInt(1, g.resizeStartW+dx)
	case handleWest, handleSouthWest:
		newW = maxInt(1, g.resizeStartW-dx)
	}

	switch g.resizeHandle {
	case handleSouth, handleSouthEast, handleSouthWest:
		newH = maxInt(1, g.resizeStartH+dy)
	}

	g.setWidgetScaleFromDrag(g.resizeStartScale, g.resizeStartW, g.resizeStartH, newW, newH)
}

func (g *Game) endResize() {
	g.isResizing = false
	g.resizeHandle = handleNone
}
