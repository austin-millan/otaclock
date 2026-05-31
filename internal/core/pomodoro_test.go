package core

import "testing"

func TestStartsInWorkPhase(t *testing.T) {
	timer := NewPomodoroTimer(DefaultPomodoroConfig())
	timer.StartOrResume(0)
	if timer.Kind() != PomodoroRunning {
		t.Fatal("expected running")
	}
	if timer.Phase() != PhaseWork {
		t.Errorf("got %q", timer.Phase())
	}
}

func TestWorkPhaseTransitionsToShortBreak(t *testing.T) {
	cfg := DefaultPomodoroConfig()
	cfg.WorkMinutes = 1
	timer := NewPomodoroTimer(cfg)
	timer.StartOrResume(0)
	phase, ended := timer.Tick(60_000)
	if !ended || phase != PhaseWork {
		t.Fatalf("expected work phase ended, got %q ended=%v", phase, ended)
	}
	if timer.Phase() != PhaseShortBreak {
		t.Errorf("got %q", timer.Phase())
	}
}

func TestFourthWorkSessionTriggersLongBreak(t *testing.T) {
	cfg := DefaultPomodoroConfig()
	cfg.WorkMinutes = 1
	cfg.ShortBreakMinutes = 1
	cfg.LongBreakEvery = 2
	timer := NewPomodoroTimer(cfg)
	timer.StartOrResume(0)

	if phase, ended := timer.Tick(60_000); !ended || phase != PhaseWork {
		t.Fatalf("step1: %q %v", phase, ended)
	}
	if phase, ended := timer.Tick(120_000); !ended || phase != PhaseShortBreak {
		t.Fatalf("step2: %q %v", phase, ended)
	}
	if phase, ended := timer.Tick(180_000); !ended || phase != PhaseWork {
		t.Fatalf("step3: %q %v", phase, ended)
	}
	if timer.Phase() != PhaseLongBreak {
		t.Errorf("got %q", timer.Phase())
	}
}

func TestPauseStopsCountdown(t *testing.T) {
	cfg := DefaultPomodoroConfig()
	cfg.WorkMinutes = 25
	timer := NewPomodoroTimer(cfg)
	timer.StartOrResume(0)
	timer.Pause()
	if _, ended := timer.Tick(60_000); ended {
		t.Error("paused timer should not advance")
	}
	if timer.Kind() != PomodoroPaused {
		t.Errorf("expected paused, got %v", timer.Kind())
	}
}

func TestStopReturnsToIdle(t *testing.T) {
	timer := NewPomodoroTimer(DefaultPomodoroConfig())
	timer.StartOrResume(0)
	timer.Stop()
	if timer.IsActive() {
		t.Error("expected inactive")
	}
	if _, ok := timer.RemainingMs(); ok {
		t.Error("expected no remaining time")
	}
}
