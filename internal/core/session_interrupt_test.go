package core_test

import (
	"testing"

	"gitlab.com/austin-millan/otaclock/internal/core"
)

func TestSessionInterrupt(t *testing.T) {
	s := core.NewSession(core.DefaultSettings())
	s.Submit("/help", 0)
	s.Interrupt()
	snap := s.Snapshot()
	if len(snap.Transcript) == 0 {
		t.Fatal("expected transcript after help + interrupt")
	}
	last := snap.Transcript[len(snap.Transcript)-1]
	if last.Content != "^C" {
		t.Fatalf("got %q, want ^C", last.Content)
	}
	if snap.IsLoading {
		t.Fatal("interrupt should clear loading")
	}
}

func TestSessionInterruptSkipsBoot(t *testing.T) {
	s := core.NewSession(core.DefaultSettings())
	s.StartBootIfNeeded(0)
	if !s.Snapshot().ShowsBootScreen {
		t.Fatal("expected boot screen")
	}
	s.Interrupt()
	snap := s.Snapshot()
	if snap.ShowsBootScreen {
		t.Fatal("interrupt should dismiss boot screen")
	}
	if len(snap.Transcript) != 0 {
		t.Fatalf("boot interrupt should leave a clean transcript, got %+v", snap.Transcript)
	}
}
