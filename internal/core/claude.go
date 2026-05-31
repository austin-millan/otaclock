package core

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	claudeDefaultAPIURL = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion    = "2023-06-01"
	claudeMaxTokens     = 1024
)

// ClaudeClient talks to the Anthropic Messages API.
type ClaudeClient struct {
	apiKey       string
	model        string
	apiURL       string
	providerName string
}

// NewClaudeClient builds a client, resolving an optional base URL override.
func NewClaudeClient(apiKey, model, baseURL string) *ClaudeClient {
	if model == "" {
		model = "claude-opus-4-5"
	}
	return &ClaudeClient{
		apiKey:       apiKey,
		model:        model,
		apiURL:       resolveEndpoint(baseURL, claudeDefaultAPIURL, "/v1/messages", "/messages"),
		providerName: fmt.Sprintf("Claude (%s)", model),
	}
}

// ProviderName returns a human-readable provider/model label.
func (c *ClaudeClient) ProviderName() string { return c.providerName }

// SendMessage sends a user turn with history and returns the assistant text.
func (c *ClaudeClient) SendMessage(ctx context.Context, content string, history []ChatMessage, systemPrompt string) (string, error) {
	messages := wireHistory(history)
	messages = append(messages, map[string]string{"role": "user", "content": content})

	body := map[string]any{
		"model":      c.model,
		"max_tokens": claudeMaxTokens,
		"system":     systemPrompt,
		"messages":   messages,
	}
	data, err := postJSON(ctx, c.apiURL, map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": claudeAPIVersion,
	}, body)
	if err != nil {
		return "", err
	}
	return claudeExtractText(data)
}

func claudeExtractText(data []byte) (string, error) {
	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil || len(parsed.Content) == 0 {
		return "", decodingError("Unexpected Claude response shape")
	}
	return parsed.Content[0].Text, nil
}
