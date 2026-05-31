package core

import "fmt"

// PomodoroConfig holds the tunable durations for the Pomodoro cycle.
type PomodoroConfig struct {
	WorkMinutes       int
	ShortBreakMinutes int
	LongBreakMinutes  int
	LongBreakEvery    int
	// PlayPhaseEndSound controls whether a sound fires when a countdown reaches zero.
	PlayPhaseEndSound bool
}

// DefaultPomodoroConfig returns the classic 25/5/15 cycle with a long break every 4 sessions.
func DefaultPomodoroConfig() PomodoroConfig {
	return PomodoroConfig{
		WorkMinutes:       25,
		ShortBreakMinutes: 5,
		LongBreakMinutes:  15,
		LongBreakEvery:    4,
		PlayPhaseEndSound: true,
	}
}

// PomodoroPhase identifies the current segment of the Pomodoro cycle.
type PomodoroPhase string

const (
	PhaseWork       PomodoroPhase = "Work"
	PhaseShortBreak PomodoroPhase = "Short break"
	PhaseLongBreak  PomodoroPhase = "Long break"
)

// PomodoroStateKind discriminates the timer's lifecycle state.
type PomodoroStateKind int

const (
	PomodoroIdle PomodoroStateKind = iota
	PomodoroRunning
	PomodoroPaused
)

// PomodoroTimer is a monotonic-clock-driven Pomodoro state machine. It is pure
// logic: callers supply the current millisecond tick to every mutating method.
type PomodoroTimer struct {
	Config PomodoroConfig

	kind                  PomodoroStateKind
	phase                 PomodoroPhase
	remainingMs           int
	completedWorkSessions int

	lastTickMs  int
	hasLastTick bool
}

// NewPomodoroTimer creates an idle timer with the given configuration.
func NewPomodoroTimer(config PomodoroConfig) PomodoroTimer {
	return PomodoroTimer{Config: config, kind: PomodoroIdle, phase: PhaseWork}
}

// Kind reports the timer's lifecycle state.
func (t PomodoroTimer) Kind() PomodoroStateKind { return t.kind }

// Phase returns the active phase (meaningful only when running or paused).
func (t PomodoroTimer) Phase() PomodoroPhase { return t.phase }

// CompletedWorkSessions reports how many work phases have finished.
func (t PomodoroTimer) CompletedWorkSessions() int { return t.completedWorkSessions }

// StartOrResume begins a fresh work phase from idle or resumes a paused timer.
func (t *PomodoroTimer) StartOrResume(nowMs int) {
	switch t.kind {
	case PomodoroIdle:
		t.kind = PomodoroRunning
		t.phase = PhaseWork
		t.remainingMs = minutesToMs(t.Config.WorkMinutes)
		t.completedWorkSessions = 0
	case PomodoroPaused:
		t.kind = PomodoroRunning
	case PomodoroRunning:
		// no-op
	}
	t.lastTickMs = nowMs
	t.hasLastTick = true
}

// Pause freezes a running countdown.
func (t *PomodoroTimer) Pause() {
	if t.kind == PomodoroRunning {
		t.kind = PomodoroPaused
	}
	t.hasLastTick = false
}

// Stop returns the timer to idle.
func (t *PomodoroTimer) Stop() {
	t.kind = PomodoroIdle
	t.hasLastTick = false
}

// Restart begins a fresh work phase regardless of the current state.
func (t *PomodoroTimer) Restart(nowMs int) {
	t.kind = PomodoroRunning
	t.phase = PhaseWork
	t.remainingMs = minutesToMs(t.Config.WorkMinutes)
	t.completedWorkSessions = 0
	t.lastTickMs = nowMs
	t.hasLastTick = true
}

// Tick advances the countdown. When a phase completes it transitions to the next
// phase and returns that just-ended phase along with true.
func (t *PomodoroTimer) Tick(nowMs int) (ended PomodoroPhase, phaseEnded bool) {
	if !t.hasLastTick {
		return "", false
	}
	delta := nowMs - t.lastTickMs
	if delta < 0 {
		delta = 0
	}
	t.lastTickMs = nowMs
	if delta == 0 {
		return "", false
	}
	if t.kind != PomodoroRunning {
		return "", false
	}

	next := t.remainingMs - delta
	if next > 0 {
		t.remainingMs = next
		return "", false
	}
	finished := t.phase
	t.advanceAfterPhaseEnded()
	return finished, true
}

// StatusText is a one-line human summary of the timer.
func (t PomodoroTimer) StatusText() string {
	switch t.kind {
	case PomodoroRunning:
		return fmt.Sprintf("Pomodoro: %s %s", t.phase, formatMs(t.remainingMs))
	case PomodoroPaused:
		return fmt.Sprintf("Pomodoro: Paused %s %s", t.phase, formatMs(t.remainingMs))
	default:
		return "Pomodoro: Idle"
	}
}

// RemainingMs returns the countdown remaining, or false when idle.
func (t PomodoroTimer) RemainingMs() (int, bool) {
	if t.kind == PomodoroIdle {
		return 0, false
	}
	return t.remainingMs, true
}

// IsActive reports whether the timer is running or paused.
func (t PomodoroTimer) IsActive() bool { return t.kind != PomodoroIdle }

func (t *PomodoroTimer) advanceAfterPhaseEnded() {
	switch t.phase {
	case PhaseWork:
		t.completedWorkSessions++
		dueLong := t.Config.LongBreakEvery > 0 && t.completedWorkSessions%t.Config.LongBreakEvery == 0
		if dueLong {
			t.phase = PhaseLongBreak
			t.remainingMs = minutesToMs(t.Config.LongBreakMinutes)
		} else {
			t.phase = PhaseShortBreak
			t.remainingMs = minutesToMs(t.Config.ShortBreakMinutes)
		}
	default: // short or long break -> back to work
		t.phase = PhaseWork
		t.remainingMs = minutesToMs(t.Config.WorkMinutes)
	}
}

func minutesToMs(minutes int) int {
	if minutes < 1 {
		minutes = 1
	}
	return minutes * 60 * 1000
}

func formatMs(ms int) string {
	if ms < 0 {
		ms = 0
	}
	total := ms / 1000
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}
