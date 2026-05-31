package termemu_test

import (
	"testing"

	"gitlab.com/austin-millan/otaclock/internal/termemu"
)

func TestVisibleCtrl(t *testing.T) {
	if termemu.VisibleCtrl(3) != termemu.CtrlC {
		t.Fatalf("VisibleCtrl(3) = %q, want %q", termemu.VisibleCtrl(3), termemu.CtrlC)
	}
	if termemu.VisibleCtrl('a') != "" {
		t.Fatal("printable rune should not render as control")
	}
}
