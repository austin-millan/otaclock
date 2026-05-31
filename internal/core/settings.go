package core

import (
	"fmt"
	"strings"
)

// Provider identifies the active LLM vendor.
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderOpenAI Provider = "openai"
)

// ProviderFromAlias normalizes user aliases (anthropic, gpt) to a canonical provider.
func ProviderFromAlias(alias string) (Provider, bool) {
	switch strings.ToLower(alias) {
	case "claude", "anthropic":
		return ProviderClaude, true
	case "openai", "gpt":
		return ProviderOpenAI, true
	default:
		return "", false
	}
}

// Settings is the unified, value-typed configuration: process defaults seeded
// from the environment plus the runtime-mutable AI connection settings.
type Settings struct {
	Use24HourClock bool

	ClaudeModelDefault string
	OpenAIModelDefault string

	AIEnabled bool
	Provider  Provider
	APIKey    string
	// Host is the API base URL. Empty means the vendor default.
	Host string
	// Model is the model id. Empty means the active provider's default.
	Model string
}

// DefaultSettings returns the baseline configuration used by tests and as a seed.
func DefaultSettings() Settings {
	return Settings{
		Use24HourClock:     false,
		ClaudeModelDefault: "claude-opus-4-5",
		OpenAIModelDefault: "gpt-4o",
		AIEnabled:          true,
		Provider:           ProviderClaude,
		APIKey:             "",
		Host:               "",
		Model:              "",
	}
}

// SettingsFromEnvironment builds settings from the provided environment map.
func SettingsFromEnvironment(env map[string]string) Settings {
	use24h := false
	if v, ok := env["OTA_24H_CLOCK"]; ok {
		use24h = v == "1" || strings.ToLower(v) == "true"
	}

	anthropicKey := env["ANTHROPIC_API_KEY"]
	openAIKey := env["OPENAI_API_KEY"]

	provider := resolveProvider(env["OTA_LLM_PROVIDER"], anthropicKey, openAIKey)

	// Prefer the configured provider's key, falling back to whichever is present.
	var key string
	switch provider {
	case ProviderClaude:
		key = firstNonEmpty(anthropicKey, openAIKey)
	case ProviderOpenAI:
		key = firstNonEmpty(openAIKey, anthropicKey)
	}

	return Settings{
		Use24HourClock:     use24h,
		ClaudeModelDefault: orDefault(env, "ANTHROPIC_MODEL", "claude-opus-4-5"),
		OpenAIModelDefault: orDefault(env, "OPENAI_MODEL", "gpt-4o"),
		AIEnabled:          key != "",
		Provider:           provider,
		APIKey:             key,
		Host:               "",
		Model:              "",
	}
}

// resolveProvider picks the active provider. An explicit OTA_LLM_PROVIDER wins;
// otherwise the provider is inferred from which API key was supplied. When only
// one provider's key is present, that provider is chosen; when both (or neither)
// are present, Claude is the default.
func resolveProvider(explicitAlias, anthropicKey, openAIKey string) Provider {
	if p, ok := ProviderFromAlias(explicitAlias); ok {
		return p
	}
	if openAIKey != "" && anthropicKey == "" {
		return ProviderOpenAI
	}
	return ProviderClaude
}

// BuildClient constructs the configured LLM client, or nil when AI is disabled
// or no API key is set.
func (s Settings) BuildClient() LLMClient {
	if !s.AIEnabled {
		return nil
	}
	trimmedKey := strings.TrimSpace(s.APIKey)
	if trimmedKey == "" {
		return nil
	}

	hostArg := strings.TrimSpace(s.Host)
	var baseURL string
	if hostArg != "" {
		baseURL = hostArg
	}
	chosenModel := strings.TrimSpace(s.Model)

	switch s.Provider {
	case ProviderOpenAI:
		resolved := chosenModel
		if resolved == "" {
			resolved = s.OpenAIModelDefault
		}
		return NewOpenAIClient(trimmedKey, resolved, baseURL)
	default:
		resolved := chosenModel
		if resolved == "" {
			resolved = s.ClaudeModelDefault
		}
		return NewClaudeClient(trimmedKey, resolved, baseURL)
	}
}

// StatusSummary renders the human-readable AI configuration for `/configure` output.
func (s Settings) StatusSummary(client LLMClient) string {
	connected := "disconnected"
	if client != nil {
		connected = client.ProviderName()
	}
	return fmt.Sprintf(`ai.enabled: %t
ai.provider: %s
ai.api_key: %s
ai.host: %s
ai.model: %s
connected: %s`,
		s.AIEnabled,
		s.Provider,
		maskAPIKey(s.APIKey),
		emptyAsDefault(s.Host),
		emptyAsDefault(s.Model),
		connected,
	)
}

func maskAPIKey(key string) string {
	k := strings.TrimSpace(key)
	if len(k) <= 8 {
		if k == "" {
			return "(not set)"
		}
		return "****"
	}
	return k[:4] + "…" + k[len(k)-4:]
}

func emptyAsDefault(s string) string {
	if s == "" {
		return "(default)"
	}
	return s
}

func orDefault(env map[string]string, key, fallback string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
