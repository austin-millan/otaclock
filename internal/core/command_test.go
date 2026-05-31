package core

import "testing"

func makeContext() CommandContext {
	return CommandContext{
		Settings: DefaultSettings(),
		Pomodoro: NewPomodoroTimer(DefaultPomodoroConfig()),
		NowMs:    0,
	}
}

func TestParserDistinguishesCommandsFromChat(t *testing.T) {
	if ParseCommand("help").Kind != CommandHelp {
		t.Error("help should parse")
	}
	if ParseCommand("/help").Kind != CommandHelp {
		t.Error("/help should parse")
	}
	p := ParseCommand("pomodoro start")
	if p.Kind != CommandPomodoro || p.State != "start" {
		t.Errorf("pomodoro start mis-parsed: %+v", p)
	}
	if ParseCommand("hello there").Kind != CommandNotACommand {
		t.Error("plain text should not be a command")
	}
	if ParseCommand("wat").Kind != CommandNotACommand {
		t.Error("unknown verbs should fall through to the LLM")
	}
}

func TestConfigureMutatesSettings(t *testing.T) {
	ctx := makeContext()
	ExecuteCommand("/configure ai.provider openai", &ctx)
	if ctx.Settings.Provider != ProviderOpenAI {
		t.Errorf("provider not set: %v", ctx.Settings.Provider)
	}
	ExecuteCommand("/configure ai.model gpt-4o-mini", &ctx)
	if ctx.Settings.Model != "gpt-4o-mini" {
		t.Errorf("model not set: %v", ctx.Settings.Model)
	}
	ExecuteCommand("/configure ai.enable false", &ctx)
	if ctx.Settings.AIEnabled {
		t.Error("ai should be disabled")
	}
}

func TestConfigureRejectsBadValues(t *testing.T) {
	ctx := makeContext()
	if ExecuteCommand("/configure ai.enable maybe", &ctx).Success {
		t.Error("bad bool should fail")
	}
	if ExecuteCommand("/configure pomodoro.short -3", &ctx).Success {
		t.Error("negative minutes should fail")
	}
}

func TestPomodoroControl(t *testing.T) {
	ctx := makeContext()
	result := ExecuteCommand("pomodoro start", &ctx)
	if !result.Success {
		t.Error("start should succeed")
	}
	if !ctx.Pomodoro.IsActive() {
		t.Error("timer should be active")
	}
	ExecuteCommand("pomodoro stop", &ctx)
	if ctx.Pomodoro.IsActive() {
		t.Error("timer should be inactive")
	}
}

func TestLoginEmitsBootEffect(t *testing.T) {
	ctx := makeContext()
	if ExecuteCommand("/login", &ctx).Effect != EffectRestartMGXBoot {
		t.Error("/login should request a boot replay")
	}
}

func TestMalformedCommandFails(t *testing.T) {
	ctx := makeContext()
	if ExecuteCommand("configure", &ctx).Success {
		t.Error("bare configure should fail")
	}
}
