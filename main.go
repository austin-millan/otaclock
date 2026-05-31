// Command otaclock2 routes to the SDL-style Ebitengine desktop widget when a
// display is available, otherwise to the headless DATAFLOW terminal.
package main

import (
	"log"
	"os"
	"runtime"

	"gitlab.com/austin-millan/otaclock/internal/gui"
	"gitlab.com/austin-millan/otaclock/internal/terminal"
)

func main() {
	if hasDisplay() {
		if err := gui.NewGame().Run(); err != nil {
			log.Fatal(err)
		}
		return
	}
	terminal.New().Run()
}

// hasDisplay reports whether a desktop session is available. On Linux this means
// an X11 or Wayland display; other platforms always have one.
func hasDisplay() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}
