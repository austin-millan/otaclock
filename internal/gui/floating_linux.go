//go:build linux

package gui

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// floatImpactNote describes what losing the floating behavior costs the user, so
// every fallback log makes the trade-off explicit.
const floatImpactNote = "the widget runs as an ordinary window (your window manager may tile, stack, or reserve space for it); all other features work normally"

func floatDebugf(format string, args ...any) {
	if os.Getenv("OTA_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[float] "+format+"\n", args...)
	}
}

// x11WindowManager floats the widget on X11 (and XWayland) by marking its window
// override-redirect, so the window manager stops managing it: no tiling, no
// decorations, no reserved layout slot. It then keeps the window stacked above
// others and routes keyboard focus to it on demand.
type x11WindowManager struct {
	conn *xgb.Conn
	win  xproto.Window
}

const (
	x11SearchTimeout  = 6 * time.Second
	x11SearchInterval = 80 * time.Millisecond
	x11RaiseInterval  = 1500 * time.Millisecond
	x11MaxTreeDepth   = 6
)

// initWindowFloating waits for Ebitengine to create the GLFW window, then
// converts it into a floating override-redirect window. Failures are silent: the
// widget still runs as an ordinary window.
func (g *Game) initWindowFloating() {
	if os.Getenv("OTA_DISABLE_FLOAT") != "" {
		return
	}
	go func() {
		wm := setupX11Floating(x11InstanceName)
		if wm == nil {
			return
		}
		g.setWindowManager(wm)
		go wm.keepOnTop()
	}()
}

func setupX11Floating(instance string) *x11WindowManager {
	// xgb prints ".Xauthority" fallback chatter to stderr even when the
	// connection ultimately succeeds; silence it and report real status below.
	xgb.Logger.SetOutput(io.Discard)

	conn, err := xgb.NewConn()
	if err != nil {
		log.Printf("otaclock: float mode unavailable (no X11 connection: %v) — %s.", err, floatImpactNote)
		return nil
	}
	root := xproto.Setup(conn).DefaultScreen(conn).Root

	var win xproto.Window
	deadline := time.Now().Add(x11SearchTimeout)
	for time.Now().Before(deadline) {
		if found, ok := findWindowByInstance(conn, root, instance, 0); ok {
			win = found
			break
		}
		time.Sleep(x11SearchInterval)
	}
	if win == 0 {
		log.Printf("otaclock: float mode unavailable (widget window not found on X server) — %s.", floatImpactNote)
		conn.Close()
		return nil
	}

	detachFromWindowManager(conn, root, win)
	hintFloatingType(conn, win)

	wm := &x11WindowManager{conn: conn, win: win}
	wm.raise()
	floatDebugf("detached window 0x%x; floating override-redirect", uint32(win))
	log.Printf("otaclock: float mode active — the widget stays on top and is excluded from window-manager tiling.")
	return wm
}

// findWindowByInstance walks the window tree looking for our WM_CLASS instance.
// The window manager may reparent the client into a decoration frame, so a
// recursive search (not just root's direct children) is required.
func findWindowByInstance(conn *xgb.Conn, win xproto.Window, instance string, depth int) (xproto.Window, bool) {
	if depth > x11MaxTreeDepth {
		return 0, false
	}
	if depth > 0 && windowInstanceMatches(conn, win, instance) {
		return win, true
	}
	tree, err := xproto.QueryTree(conn, win).Reply()
	if err != nil || tree == nil {
		return 0, false
	}
	for _, child := range tree.Children {
		if found, ok := findWindowByInstance(conn, child, instance, depth+1); ok {
			return found, true
		}
	}
	return 0, false
}

func windowInstanceMatches(conn *xgb.Conn, win xproto.Window, instance string) bool {
	reply, err := xproto.GetProperty(conn, false, win, xproto.AtomWmClass, xproto.AtomString, 0, 256).Reply()
	if err != nil || reply == nil || len(reply.Value) == 0 {
		return false
	}
	for _, part := range strings.Split(string(reply.Value), "\x00") {
		if part == instance {
			return true
		}
	}
	return false
}

// detachFromWindowManager re-maps the window as an override-redirect child of
// the root window. Most window managers reparent a managed client into a
// decoration frame; remapping the client while it is still framed leaves it in
// an inconsistent state where it neither receives input focus nor mouse clicks
// reliably (clicks fall through to the windows behind it). Reparenting it back
// to the root at its current screen position first guarantees it maps as a
// normal top-level window that the X server routes input to.
//
// The override-redirect attribute only takes effect on map, so the window is
// briefly unmapped and re-mapped; the WM then unmanages it (no tiling, no frame).
func detachFromWindowManager(conn *xgb.Conn, root, win xproto.Window) {
	rootX, rootY, haveAbsolute := absoluteWindowPosition(conn, root, win)

	xproto.UnmapWindow(conn, win)
	if haveAbsolute {
		xproto.ReparentWindow(conn, win, root, rootX, rootY)
	}
	xproto.ChangeWindowAttributes(conn, win, xproto.CwOverrideRedirect, []uint32{1})
	xproto.MapWindow(conn, win)
	syncConn(conn)
}

// absoluteWindowPosition returns win's top-left corner in root coordinates,
// resolving any decoration-frame offset the window manager added.
func absoluteWindowPosition(conn *xgb.Conn, root, win xproto.Window) (x, y int16, ok bool) {
	trans, err := xproto.TranslateCoordinates(conn, win, root, 0, 0).Reply()
	if err != nil || trans == nil {
		return 0, 0, false
	}
	return int16(trans.DstX), int16(trans.DstY), true
}

// hintFloatingType also advertises UTILITY + ABOVE so managers that honor EWMH
// hints (or that ignore override-redirect) still float and stack the widget.
func hintFloatingType(conn *xgb.Conn, win xproto.Window) {
	windowType := internAtom(conn, "_NET_WM_WINDOW_TYPE")
	utility := internAtom(conn, "_NET_WM_WINDOW_TYPE_UTILITY")
	if windowType != 0 && utility != 0 {
		setAtomProperty(conn, win, windowType, utility)
	}

	state := internAtom(conn, "_NET_WM_STATE")
	above := internAtom(conn, "_NET_WM_STATE_ABOVE")
	if state != 0 && above != 0 {
		setAtomProperty(conn, win, state, above)
	}
	syncConn(conn)
}

func setAtomProperty(conn *xgb.Conn, win xproto.Window, property, value xproto.Atom) {
	data := make([]byte, 4)
	binary.NativeEndian.PutUint32(data, uint32(value))
	xproto.ChangeProperty(conn, xproto.PropModeReplace, win, property, xproto.AtomAtom, 32, 1, data)
}

func internAtom(conn *xgb.Conn, name string) xproto.Atom {
	reply, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
	if err != nil || reply == nil {
		return 0
	}
	return reply.Atom
}

// syncConn forces queued requests to the server via a cheap round-trip.
func syncConn(conn *xgb.Conn) {
	_, _ = xproto.GetInputFocus(conn).Reply()
}

// FocusAndRaise raises the widget and directs keyboard input to it.
func (m *x11WindowManager) FocusAndRaise() {
	m.raise()
	xproto.SetInputFocus(m.conn, xproto.InputFocusParent, m.win, xproto.TimeCurrentTime)
	syncConn(m.conn)
}

func (m *x11WindowManager) raise() {
	xproto.ConfigureWindow(m.conn, m.win, xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove})
	syncConn(m.conn)
}

// keepOnTop periodically restacks the widget so it stays above other windows.
func (m *x11WindowManager) keepOnTop() {
	ticker := time.NewTicker(x11RaiseInterval)
	defer ticker.Stop()
	for range ticker.C {
		m.raise()
	}
}
