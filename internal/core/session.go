package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Snapshot is an immutable view of the session for front-ends to render.
type Snapshot struct {
	Transcript     []ChatMessage
	IsLoading      bool
	Clock          ClockSnapshot
	ClockDisplay   string
	Use24HourClock bool
	ProviderName   string
	Pomodoro       PomodoroTimer
	// ShowsBootScreen is true while front-ends should show the MGX boot log.
	ShowsBootScreen bool
}

// Session is the single source of truth shared by every front-end. Domain state
// is guarded by a mutex so the synchronous render loop and detached LLM
// goroutines can touch it safely; front-ends read a consistent picture via
// Snapshot.
type Session struct {
	// OnPomodoroPhaseEnded is invoked (on the ticking goroutine) when a phase ends.
	OnPomodoroPhaseEnded func(PomodoroPhase)

	mu sync.Mutex

	clockService ClockService

	settings     Settings
	llmClient    LLMClient
	providerName string
	transcript   []ChatMessage
	llmHistory   []ChatMessage
	loading      bool
	clockSnap    ClockSnapshot
	pomodoro     PomodoroTimer
	boot         BootPlayback
	// showsBootScreen stays true until the user submits their first line.
	showsBootScreen bool
}

// NewSession constructs a session seeded from the supplied settings.
func NewSession(settings Settings) *Session {
	s := &Session{
		settings:        settings,
		showsBootScreen: true,
		pomodoro:        NewPomodoroTimer(DefaultPomodoroConfig()),
	}
	s.clockSnap = s.clockService.Snapshot()
	s.llmClient = settings.BuildClient()
	s.providerName = "offline"
	if s.llmClient != nil {
		s.providerName = s.llmClient.ProviderName()
	}
	return s
}

// NewSessionFromEnvironment constructs a session using the process environment.
func NewSessionFromEnvironment() *Session {
	return NewSession(SettingsFromEnvironment(environMap()))
}

func environMap() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	return env
}

// Snapshot returns a consistent, copyable view of the session.
func (s *Session) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.makeSnapshotLocked()
}

// ProviderName returns the connected provider label (or "offline").
func (s *Session) ProviderName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.providerName
}

// Tick advances the wall clock and Pomodoro timer. nowMs is a monotonic tick.
func (s *Session) Tick(nowMs uint32) {
	var endedPhase PomodoroPhase
	hasEnded := false

	s.mu.Lock()
	s.clockSnap = s.clockService.Snapshot()
	if phase, ended := s.pomodoro.Tick(int(nowMs)); ended {
		endedPhase = phase
		hasEnded = true
	}
	s.mu.Unlock()

	if hasEnded {
		if s.OnPomodoroPhaseEnded != nil {
			s.OnPomodoroPhaseEnded(endedPhase)
		}
		s.sendPomodoroPhaseEnd(endedPhase)
	}
}

// Submit routes a line of input: a known command first, otherwise chat.
func (s *Session) Submit(line string, nowMs uint32) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if ParseCommand(trimmed).Kind != CommandNotACommand {
		s.handleCommand(trimmed, nowMs)
	} else {
		s.sendChat(trimmed)
	}
}

// Interrupt handles a Ctrl-C: while the boot screen is playing it simply skips
// to the ready prompt; otherwise it echoes a "^C" line. In both cases any
// in-flight request is no longer awaited.
func (s *Session) Interrupt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loading = false
	if s.showsBootScreen {
		s.showsBootScreen = false
		return
	}
	s.transcript = append(s.transcript, NewChatMessage(RoleSystem, "^C"))
}

// ClearTranscript clears the visible transcript and the LLM context.
func (s *Session) ClearTranscript() {
	s.mu.Lock()
	s.transcript = nil
	s.llmHistory = nil
	s.mu.Unlock()
}

// StartBootIfNeeded begins boot playback the first time it is requested.
func (s *Session) StartBootIfNeeded(nowMs uint32) {
	s.mu.Lock()
	s.boot.StartIfNeeded(nowMs)
	s.mu.Unlock()
}

// RestartBoot replays the boot sequence from the beginning.
func (s *Session) RestartBoot(nowMs uint32) {
	s.mu.Lock()
	s.boot.Restart(nowMs)
	s.mu.Unlock()
}

// BootFrame returns the boot playback frame at the supplied tick.
func (s *Session) BootFrame(nowMs uint32) BootFrame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.boot.Frame(nowMs)
}

// IsBootComplete reports whether the boot sequence has fully printed.
func (s *Session) IsBootComplete(nowMs uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.boot.IsComplete(nowMs)
}

// Pomodoro control surface (menu / direct UI).

func (s *Session) PomodoroStartOrResume(nowMs uint32) {
	s.mutatePomodoro(func(p *PomodoroTimer) { p.StartOrResume(int(nowMs)) })
}
func (s *Session) PomodoroPause() { s.mutatePomodoro(func(p *PomodoroTimer) { p.Pause() }) }
func (s *Session) PomodoroStop()  { s.mutatePomodoro(func(p *PomodoroTimer) { p.Stop() }) }
func (s *Session) PomodoroRestart(nowMs uint32) {
	s.mutatePomodoro(func(p *PomodoroTimer) { p.Restart(int(nowMs)) })
}
func (s *Session) TogglePomodoroSound() {
	s.mutatePomodoro(func(p *PomodoroTimer) { p.Config.PlayPhaseEndSound = !p.Config.PlayPhaseEndSound })
}

// PomodoroConfig returns the current Pomodoro configuration.
func (s *Session) PomodoroConfig() PomodoroConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pomodoro.Config
}

// UpdatePomodoroConfig applies a transform to the Pomodoro configuration.
func (s *Session) UpdatePomodoroConfig(transform func(*PomodoroConfig)) {
	s.mu.Lock()
	transform(&s.pomodoro.Config)
	s.mu.Unlock()
}

// --- internals ---

func (s *Session) handleCommand(line string, nowMs uint32) {
	s.mu.Lock()
	ctx := CommandContext{Settings: s.settings, Pomodoro: s.pomodoro, NowMs: int(nowMs)}
	result := ExecuteCommand(line, &ctx)
	s.settings = ctx.Settings
	s.pomodoro = ctx.Pomodoro
	s.rebuildClientLocked()
	if result.Effect == EffectRestartMGXBoot {
		s.showsBootScreen = true
		s.boot.Restart(nowMs)
	} else {
		s.showsBootScreen = false
		s.transcript = append(s.transcript, NewChatMessage(RoleUser, line))
		if len(result.Lines) > 0 {
			s.transcript = append(s.transcript, NewChatMessage(RoleSystem, strings.Join(result.Lines, "\n")))
		}
	}
	s.mu.Unlock()
}

func (s *Session) sendChat(text string) {
	s.mu.Lock()
	if s.loading {
		s.mu.Unlock()
		return
	}
	s.showsBootScreen = false
	s.loading = true
	s.transcript = append(s.transcript, NewChatMessage(RoleUser, text))
	s.llmHistory = append(s.llmHistory, NewChatMessage(RoleUser, text))
	client := s.llmClient
	history := cloneMessages(s.llmHistory)
	s.mu.Unlock()

	if client == nil {
		s.finishWith(nil, NoAPIKeyMessage)
		return
	}

	go func() {
		reply, err := SendMessageDefaultPrompt(context.Background(), client, text, history)
		if err != nil {
			s.finishWith(err, "")
			return
		}
		s.completeReply(reply)
	}()
}

func (s *Session) sendPomodoroPhaseEnd(phase PomodoroPhase) {
	s.mu.Lock()
	if s.loading {
		s.mu.Unlock()
		return
	}
	client := s.llmClient
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.loading = true
	history := cloneMessages(s.llmHistory)
	s.mu.Unlock()

	systemPrompt := PomodoroPhaseEndSystemPrompt(string(phase))
	go func() {
		reply, err := client.SendMessage(context.Background(), PomodoroPhaseEndUserMessage, history, systemPrompt)
		if err != nil {
			s.mu.Lock()
			s.loading = false
			s.mu.Unlock()
			return
		}
		s.completeReply(reply)
	}()
}

func (s *Session) completeReply(reply string) {
	s.mu.Lock()
	s.llmHistory = append(s.llmHistory, NewChatMessage(RoleAssistant, reply))
	s.transcript = append(s.transcript, NewChatMessage(RoleAssistant, reply))
	s.loading = false
	s.mu.Unlock()
}

func (s *Session) finishWith(err error, fallback string) {
	s.mu.Lock()
	text := fallback
	if text == "" {
		msg := "Unknown error"
		if err != nil {
			msg = err.Error()
		}
		text = fmt.Sprintf("Snake, we've lost the signal. %s", msg)
	}
	s.transcript = append(s.transcript, NewChatMessage(RoleAssistant, text))
	s.loading = false
	s.mu.Unlock()
}

func (s *Session) makeSnapshotLocked() Snapshot {
	display := fmt.Sprintf("%s %s", s.clockSnap.FormattedTime12h(), s.clockSnap.AMPM())
	if s.settings.Use24HourClock {
		display = s.clockSnap.FormattedTime24h()
	}
	return Snapshot{
		Transcript:      cloneMessages(s.transcript),
		IsLoading:       s.loading,
		Clock:           s.clockSnap,
		ClockDisplay:    display,
		Use24HourClock:  s.settings.Use24HourClock,
		ProviderName:    s.providerName,
		Pomodoro:        s.pomodoro,
		ShowsBootScreen: s.showsBootScreen,
	}
}

func (s *Session) rebuildClientLocked() {
	s.llmClient = s.settings.BuildClient()
	if s.llmClient != nil {
		s.providerName = s.llmClient.ProviderName()
	} else {
		s.providerName = "offline"
	}
}

func (s *Session) mutatePomodoro(transform func(*PomodoroTimer)) {
	s.mu.Lock()
	transform(&s.pomodoro)
	s.mu.Unlock()
}

func cloneMessages(in []ChatMessage) []ChatMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]ChatMessage, len(in))
	copy(out, in)
	return out
}
