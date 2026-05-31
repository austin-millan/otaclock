package core

import (
	"strings"
	"testing"
)

func TestEnvironmentPrefersConfiguredProviderKey(t *testing.T) {
	settings := SettingsFromEnvironment(map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant",
		"OPENAI_API_KEY":    "sk-openai",
		"OTA_LLM_PROVIDER":  "openai",
	})
	if settings.Provider != ProviderOpenAI {
		t.Errorf("provider: %v", settings.Provider)
	}
	if settings.APIKey != "sk-openai" {
		t.Errorf("key: %v", settings.APIKey)
	}
	if !settings.AIEnabled {
		t.Error("ai should be enabled")
	}
}

func TestEnvironmentFallsBackToOtherProviderKey(t *testing.T) {
	settings := SettingsFromEnvironment(map[string]string{
		"OPENAI_API_KEY":   "sk-openai",
		"OTA_LLM_PROVIDER": "claude",
	})
	if settings.Provider != ProviderClaude {
		t.Errorf("provider: %v", settings.Provider)
	}
	if settings.APIKey != "sk-openai" {
		t.Errorf("key: %v", settings.APIKey)
	}
}

func TestProviderDefaultsToSuppliedKey(t *testing.T) {
	openAIOnly := SettingsFromEnvironment(map[string]string{"OPENAI_API_KEY": "sk-openai"})
	if openAIOnly.Provider != ProviderOpenAI {
		t.Errorf("openai key alone should default to openai, got %v", openAIOnly.Provider)
	}
	if openAIOnly.APIKey != "sk-openai" {
		t.Errorf("key: %v", openAIOnly.APIKey)
	}

	claudeOnly := SettingsFromEnvironment(map[string]string{"ANTHROPIC_API_KEY": "sk-ant"})
	if claudeOnly.Provider != ProviderClaude {
		t.Errorf("anthropic key alone should default to claude, got %v", claudeOnly.Provider)
	}
	if claudeOnly.APIKey != "sk-ant" {
		t.Errorf("key: %v", claudeOnly.APIKey)
	}
}

func TestProviderDefaultsToClaudeWhenBothKeysPresent(t *testing.T) {
	settings := SettingsFromEnvironment(map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant",
		"OPENAI_API_KEY":    "sk-openai",
	})
	if settings.Provider != ProviderClaude {
		t.Errorf("both keys should default to claude, got %v", settings.Provider)
	}
	if settings.APIKey != "sk-ant" {
		t.Errorf("key: %v", settings.APIKey)
	}
}

func TestDisabledWithoutKey(t *testing.T) {
	settings := SettingsFromEnvironment(map[string]string{})
	if settings.AIEnabled {
		t.Error("ai should be disabled")
	}
	if settings.BuildClient() != nil {
		t.Error("client should be nil")
	}
}

func Test24HourClockFlag(t *testing.T) {
	if !SettingsFromEnvironment(map[string]string{"OTA_24H_CLOCK": "true"}).Use24HourClock {
		t.Error("true should enable 24h")
	}
	if !SettingsFromEnvironment(map[string]string{"OTA_24H_CLOCK": "1"}).Use24HourClock {
		t.Error("1 should enable 24h")
	}
	if SettingsFromEnvironment(map[string]string{}).Use24HourClock {
		t.Error("default should be 12h")
	}
}

func TestBuildClientHonorsProvider(t *testing.T) {
	settings := Settings{AIEnabled: true, Provider: ProviderClaude, APIKey: "k", ClaudeModelDefault: "claude-opus-4-5", OpenAIModelDefault: "gpt-4o"}
	if c := settings.BuildClient(); c == nil || !strings.HasPrefix(c.ProviderName(), "Claude") {
		t.Error("expected Claude client")
	}
	settings.Provider = ProviderOpenAI
	if c := settings.BuildClient(); c == nil || !strings.HasPrefix(c.ProviderName(), "OpenAI") {
		t.Error("expected OpenAI client")
	}
}

func TestProviderAliasNormalization(t *testing.T) {
	if p, ok := ProviderFromAlias("anthropic"); !ok || p != ProviderClaude {
		t.Error("anthropic -> claude")
	}
	if p, ok := ProviderFromAlias("gpt"); !ok || p != ProviderOpenAI {
		t.Error("gpt -> openai")
	}
	if _, ok := ProviderFromAlias("bogus"); ok {
		t.Error("bogus should fail")
	}
}
