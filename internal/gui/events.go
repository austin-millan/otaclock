package gui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

const doubleClickMs = 350

func (g *Game) handleInput(now uint32) {
	g.handleTextInput()
	g.handleKeyboard(now)
	g.handleMouse(now)
}

// --- text input ---

func (g *Game) handleTextInput() {
	chars := ebiten.AppendInputChars(nil)
	if len(chars) == 0 {
		return
	}
	if g.menuVisible {
		g.handleMenuTextInput(chars)
		return
	}
	if g.uiMode == modeChat {
		g.inputText += string(chars)
		g.recomputeInputLayout()
	}
}

// --- keyboard ---

// Key-repeat timing for held keys, in 60 TPS ticks: an initial delay before the
// first repeat, then one delete per interval (~30/sec) thereafter.
const (
	keyRepeatDelayTicks    = 28
	keyRepeatIntervalTicks = 2
)

func (g *Game) handleKeyboard(now uint32) {
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		g.handleKeyDown(k, now)
	}
	// Backspace is driven by key-repeat so holding it clears text continuously.
	if repeatingKeyPressed(ebiten.KeyBackspace) {
		g.handleBackspace()
	}
}

// repeatingKeyPressed reports true on the initial press and then on a steady
// cadence while the key is held, mirroring OS keyboard auto-repeat.
func repeatingKeyPressed(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true
	}
	return d >= keyRepeatDelayTicks && (d-keyRepeatDelayTicks)%keyRepeatIntervalTicks == 0
}

// handleBackspace deletes one character from whichever field currently has
// focus: a Pomodoro config field when the menu is open, otherwise the chat input.
func (g *Game) handleBackspace() {
	if g.menuVisible {
		if g.menuScreen == menuPomodoroConfig && g.menuFocusField < 4 {
			g.mutateFocusedField(func(s string) string {
				if len(s) > 0 {
					return s[:len(s)-1]
				}
				return s
			})
		}
		return
	}
	if g.uiMode != modeChat {
		return
	}
	if g.inputText != "" {
		runes := []rune(g.inputText)
		g.inputText = string(runes[:len(runes)-1])
	}
	g.recomputeInputLayout()
}

func (g *Game) handleKeyDown(k ebiten.Key, now uint32) {
	if g.menuVisible {
		g.handleMenuKey(k)
		return
	}
	if k == ebiten.KeyEscape {
		g.quit = true
		return
	}
	if g.handleZoomHotkeys(k) {
		return
	}
	if g.uiMode != modeChat {
		return
	}

	switch k {
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		g.submitInput(now)
	case ebiten.KeyPageUp:
		g.scrollOffset += g.mgxResponseVisibleLineCount()
	case ebiten.KeyPageDown:
		g.scrollOffset = maxInt(0, g.scrollOffset-g.mgxResponseVisibleLineCount())
	case ebiten.KeyHome:
		g.scrollOffset = 1 << 30
	case ebiten.KeyEnd:
		g.scrollOffset = 0
	case ebiten.KeyUp:
		g.inputScrollLines++
	case ebiten.KeyDown:
		g.inputScrollLines = maxInt(0, g.inputScrollLines-1)
	}
}

func (g *Game) handleZoomHotkeys(k ebiten.Key) bool {
	if !ebiten.IsKeyPressed(ebiten.KeyControl) {
		return false
	}
	isPlus := k == ebiten.KeyEqual || k == ebiten.KeyNumpadAdd
	isMinus := k == ebiten.KeyMinus || k == ebiten.KeyNumpadSubtract
	if !isPlus && !isMinus {
		return false
	}
	step := widgetScaleStep
	if isMinus {
		step = -widgetScaleStep
	}
	g.applyWidgetScale(g.widgetScale + step)
	return true
}

func (g *Game) submitInput(now uint32) {
	trimmed := strings.TrimSpace(g.inputText)
	g.inputText = ""
	g.inputScrollLines = 0
	g.inputH = g.layout.InputHeight()
	g.applyChatWindowHeight(g.appH)
	if trimmed == "" {
		return
	}
	if strings.ToLower(trimmed) == "exit" {
		g.setChatPanel(false)
		return
	}
	g.session.Submit(trimmed, now)
	g.scrollOffset = 0
	g.syncPomodoroStringsFromConfig()
}

// --- mouse ---

func (g *Game) handleMouse(now uint32) {
	if _, yoff := ebiten.Wheel(); yoff != 0 && g.uiMode == modeChat {
		g.scrollOffset = maxInt(0, g.scrollOffset+int(yoff)*2)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.handleMouseDown(ebiten.MouseButtonLeft, now)
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.handleMouseDown(ebiten.MouseButtonRight, now)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) ||
		inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		g.isDragging = false
		g.endResize()
	}

	g.handleMouseMotion()
}

func (g *Game) handleMouseDown(button ebiten.MouseButton, now uint32) {
	cx, cy := ebiten.CursorPosition()

	// Override-redirect windows aren't focused by the WM, so grab focus and
	// raise on any click to keep the chat input usable and the widget on top.
	g.requestRefocus()

	if g.menuVisible {
		ox := g.menuOriginX()
		if cx >= ox && cx < ox+g.menuW() && cy >= 0 && cy < g.menuH() {
			g.handleMenuClick(cx-ox, cy, button)
			return
		}
	}

	if button == ebiten.MouseButtonRight {
		if g.isPointInSpriteRegion(cx, cy) {
			g.toggleMenu()
		}
		return
	}

	if g.menuVisible {
		g.hideMenu()
	}
	if button != ebiten.MouseButtonLeft {
		return
	}

	if handle := g.resizeHandleAt(cx, cy); handle != handleNone {
		g.beginResize(handle)
		return
	}

	clicks := g.registerClick(cx, cy, now)

	snap := g.session.Snapshot()
	if g.isPointInSpriteRegion(cx, cy) && snap.Pomodoro.IsActive() &&
		g.spriteRenderer != nil &&
		g.spriteRenderer.IsPointInPomodoroOverlay(cx-g.spriteOriginX(), cy) {
		g.handlePomodoroClick(clicks, snap.Pomodoro, now)
		return
	}

	if clicks == 2 {
		if g.isPointInSpriteRegion(cx, cy) {
			g.toggleChatPanel()
		}
	} else {
		g.beginDrag(cx, cy)
	}
}

// registerClick reports the click count (1 or 2) using time + proximity.
func (g *Game) registerClick(cx, cy int, now uint32) int {
	clicks := 1
	if now-g.lastLeftClickMs <= doubleClickMs &&
		absInt(cx-g.lastClickX) <= 6 && absInt(cy-g.lastClickY) <= 6 {
		clicks = 2
	}
	g.lastLeftClickMs = now
	g.lastClickX = cx
	g.lastClickY = cy
	return clicks
}

func (g *Game) handlePomodoroClick(clicks int, pomo core.PomodoroTimer, now uint32) {
	if clicks >= 2 {
		g.session.PomodoroRestart(now)
		return
	}
	switch pomo.Kind() {
	case core.PomodoroRunning:
		g.session.PomodoroPause()
	case core.PomodoroPaused:
		g.session.PomodoroStartOrResume(now)
	}
}

func (g *Game) beginDrag(cx, cy int) {
	g.isDragging = true
	g.dragOffsetX = cx
	g.dragOffsetY = cy
}

func (g *Game) handleMouseMotion() {
	if g.isResizing {
		g.updateResizeDrag()
		return
	}
	if g.isDragging {
		wx, wy := ebiten.WindowPosition()
		cx, cy := ebiten.CursorPosition()
		ebiten.SetWindowPosition(wx+cx-g.dragOffsetX, wy+cy-g.dragOffsetY)
		// Moving the override-redirect window can also drop focus; keep
		// re-asserting it throughout the drag and briefly after it ends.
		g.requestRefocus()
		return
	}

	cx, cy := ebiten.CursorPosition()
	spriteX := cx - g.spriteOriginX()
	if g.isBlinking || g.currentAnimFrame != 0 ||
		spriteX < 0 || spriteX >= g.layout.WindowWidth() || cy >= g.layout.SpriteHeight() {
		return
	}

	pivotX := g.spriteOriginX() + 67*g.layout.SpriteScale()
	pivotY := 47 * g.layout.SpriteScale()
	isTop := cy < pivotY
	isRight := cx > pivotX
	switch {
	case isTop && isRight:
		g.currentEyeState = 4
	case isTop:
		g.currentEyeState = 2
	case isRight:
		g.currentEyeState = 3
	default:
		g.currentEyeState = 1
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
