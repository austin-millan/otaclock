package gui

// X11 WM_CLASS identifiers; the floating logic locates our own window by these.
const (
	x11ClassName    = "OtaClock2"
	x11InstanceName = "otaclock2"
)

// windowManager applies platform window behaviors that Ebitengine does not
// expose: keeping the widget above other windows and grabbing keyboard focus.
// On window managers that tile normal windows, the implementation also detaches
// the widget from management so it floats instead of being tiled.
type windowManager interface {
	// FocusAndRaise raises the widget to the top and gives it keyboard focus.
	FocusAndRaise()
}

func (g *Game) setWindowManager(wm windowManager) {
	g.wmMu.Lock()
	g.wm = wm
	g.wmMu.Unlock()
}

// focusAndRaise is a no-op until the platform window manager has been wired up.
func (g *Game) focusAndRaise() {
	g.wmMu.Lock()
	wm := g.wm
	g.wmMu.Unlock()
	if wm != nil {
		wm.FocusAndRaise()
	}
}
