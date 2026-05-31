//go:build !linux

package gui

// initWindowFloating is a no-op off Linux. On Windows and macOS the borderless,
// floating, skip-taskbar window flags set in Run already keep the widget on top
// and out of normal window management.
func (g *Game) initWindowFloating() {}
