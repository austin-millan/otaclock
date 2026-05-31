// Package gui implements the Ebitengine desktop widget front-end: a transparent,
// borderless, always-on-top Otacon clock with a Metal Gear Solid-style DATAFLOW
// chat panel and an in-window Pomodoro tools menu.
package gui

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

type uiMode int

const (
	modeCompact uiMode = iota
	modeChat
)

// Game is the Ebitengine front-end. It owns the render loop and binds the
// window to a shared core.Session.
type Game struct {
	session *core.Session
	start   time.Time

	spriteRenderer *SpriteRenderer
	fonts          *FontSet
	sound          *SoundPlayer

	// Geometry. appW/appH are the content dimensions; the menu strip (when
	// visible) is appended to the right, producing the full window size.
	widgetScale float64
	layout      LayoutMetrics
	appW        int
	appH        int
	responseH   int
	inputH      int
	uiMode      uiMode

	lastWindowW int
	lastWindowH int

	inputText        string
	inputScrollLines int
	scrollOffset     int
	prevMessageCount int

	quit bool

	// Dragging.
	isDragging  bool
	dragOffsetX int
	dragOffsetY int

	// Resizing.
	isResizing        bool
	resizeHandle      resizeHandle
	resizeStartScale  float64
	resizeStartW      int
	resizeStartH      int
	resizeStartMouseX int
	resizeStartMouseY int
	resizeStartWinX   int
	resizeStartWinY   int

	// Click tracking (Ebiten has no native double-click).
	lastLeftClickMs uint32
	lastClickX      int
	lastClickY      int

	// Sprite animation.
	currentEyeState  int
	currentAnimFrame int
	lastAnimTick     uint32
	isBlinking       bool
	lastBlinkTick    uint32
	nextBlinkTick    uint32
	isTalking        bool
	talkEndTick      uint32
	prevIsLoading    bool

	// Tools menu (in-window overlay).
	menuVisible    bool
	menuScreen     menuScreen
	menuFocusField int
	pomoWorkStr    string
	pomoShortStr   string
	pomoLongStr    string
	pomoEveryStr   string

	// Platform window manager (float-on-top / focus), wired asynchronously.
	wmMu sync.Mutex
	wm   windowManager

	// refocusFrames counts down the frames over which input focus is re-asserted
	// after an interaction, to win focus races with the window manager.
	refocusFrames int
}

// NewGame constructs the front-end bound to a fresh session from the environment.
func NewGame() *Game {
	g := &Game{
		session:         core.NewSessionFromEnvironment(),
		start:           time.Now(),
		widgetScale:     1.0,
		uiMode:          modeCompact,
		currentEyeState: 1,
		pomoWorkStr:     "25",
		pomoShortStr:    "5",
		pomoLongStr:     "15",
		pomoEveryStr:    "4",
	}
	g.layout = newLayout(g.widgetScale)
	g.appW = g.layout.WindowWidth()
	g.appH = g.layout.CompactWindowHeight()
	g.responseH = g.layout.DefaultResponseHeight()
	g.inputH = g.layout.InputHeight()
	return g
}

// Run sets up the transparent floating window and blocks until dismissed.
func (g *Game) Run() error {
	g.spriteRenderer = NewSpriteRenderer(g.layout.SpriteScale())
	if fs, err := NewFontSet(g.layout.DefaultFontNormPt(), g.layout.DefaultFontSmallPt()); err == nil {
		g.fonts = fs
	}
	if sp, err := NewSoundPlayer(); err == nil {
		g.sound = sp
	}

	g.session.OnPomodoroPhaseEnded = func(core.PomodoroPhase) {
		if g.session.PomodoroConfig().PlayPhaseEndSound && g.sound != nil {
			g.sound.Play()
		}
	}

	g.scheduleNextBlink()

	ebiten.SetWindowTitle("OtaClock2")
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	// Keep the clock ticking even when another window holds focus, and tolerate
	// the brief unmap/remap the X11 floating path performs.
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowSize(g.appW, g.appH)
	g.lastWindowW, g.lastWindowH = g.appW, g.appH
	g.centerWindow()

	g.initWindowFloating()

	op := &ebiten.RunGameOptions{
		ScreenTransparent: true,
		SkipTaskbar:       true,
		X11ClassName:      x11ClassName,
		X11InstanceName:   x11InstanceName,
	}
	return ebiten.RunGameWithOptions(g, op)
}

func (g *Game) centerWindow() {
	mw, mh := ebiten.Monitor().Size()
	if mw <= 0 || mh <= 0 {
		return
	}
	ebiten.SetWindowPosition((mw-g.appW)/2, (mh-g.appH)/2)
}

// nowMs returns a monotonic millisecond tick.
func (g *Game) nowMs() uint32 {
	return uint32(time.Since(g.start).Milliseconds())
}

// Update advances input, the clock/timers, and animation state once per tick.
func (g *Game) Update() error {
	now := g.nowMs()
	g.handleInput(now)
	g.session.Tick(now)
	snap := g.session.Snapshot()
	g.updateAnimations(snap)
	g.syncWindowSize()
	g.serviceRefocus()
	if g.quit {
		return ebiten.Termination
	}
	return nil
}

// refocusBurstFrames is how many consecutive frames input focus is re-asserted
// after an interaction (~150ms at 60fps), long enough to beat a window manager
// that reclaims focus immediately after our first SetInputFocus.
const refocusBurstFrames = 8

// requestRefocus schedules a burst of input-focus re-assertions. It is triggered
// by interactions with the widget (click, drag, resize) so the chat input stays
// typable; it deliberately does not fire on hover, so focus can still move to
// other apps when the user clicks them.
func (g *Game) requestRefocus() { g.refocusFrames = refocusBurstFrames }

func (g *Game) serviceRefocus() {
	if g.refocusFrames <= 0 {
		return
	}
	g.refocusFrames--
	g.focusAndRaise()
}

// Draw renders the widget, optional chat panel, and optional tools menu.
func (g *Game) Draw(screen *ebiten.Image) {
	snap := g.session.Snapshot()

	now := g.nowMs()
	showDots := (now/500)%2 == 0
	eyeState := g.currentEyeState
	if g.isBlinking {
		eyeState = 0
	}
	alarmDigits := pomodoroAlarmDigits(snap)

	if g.spriteRenderer != nil {
		g.spriteRenderer.Render(
			screen, g.spriteOriginX(), 0,
			snap.Clock, eyeState, g.currentAnimFrame,
			!snap.Use24HourClock, showDots,
			alarmDigits, alarmDigits != nil && showDots,
		)
	}

	if g.uiMode == modeChat {
		fillRect(screen, 0, g.layout.ChatPanelY(), g.appW, 1, Theme.TerminalBorder)
		g.renderResponsePanel(screen, snap)
		g.renderInputBar(screen, snap)
	}

	if g.menuVisible {
		g.renderMenu(screen)
	}
}

// Layout reports the logical screen size (content plus the menu strip).
func (g *Game) Layout(_, _ int) (int, int) {
	return g.screenW(), g.screenH()
}

// --- geometry helpers ---

func (g *Game) screenW() int {
	if g.menuVisible {
		return g.appW + g.layout.Scaled(8) + g.menuW()
	}
	return g.appW
}

func (g *Game) screenH() int {
	if g.menuVisible {
		return maxInt(g.appH, g.menuH())
	}
	return g.appH
}

func (g *Game) syncWindowSize() {
	w, h := g.screenW(), g.screenH()
	if w != g.lastWindowW || h != g.lastWindowH {
		ebiten.SetWindowSize(w, h)
		g.lastWindowW, g.lastWindowH = w, h
		// Resizing the override-redirect window on X11 can silently drop input
		// focus, leaving the chat input dead after the open/close toggle.
		g.requestRefocus()
	}
}

func (g *Game) spriteOriginX() int {
	if g.uiMode == modeChat {
		return g.appW - g.layout.WindowWidth()
	}
	return 0
}

func (g *Game) isPointInSpriteRegion(x, y int) bool {
	return y < g.layout.SpriteHeight() && x >= g.spriteOriginX() && x < g.spriteOriginX()+g.layout.WindowWidth()
}

func (g *Game) lineHeight() int {
	if g.fonts != nil {
		return g.fonts.LineHeight
	}
	return 14
}

func pomodoroAlarmDigits(snap core.Snapshot) *[4]int {
	if !snap.Pomodoro.IsActive() {
		return nil
	}
	remaining, ok := snap.Pomodoro.RemainingMs()
	if !ok {
		return nil
	}
	total := remaining
	if total < 0 {
		total = 0
	}
	total /= 1000
	m := total / 60
	if m > 99 {
		m = 99
	}
	s := total % 60
	return &[4]int{m / 10, m % 10, s / 10, s % 10}
}
