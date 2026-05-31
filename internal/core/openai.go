package core

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	openAIDefaultAPIURL = "https://api.openai.com/v1/chat/completions"
	openAIMaxTokens     = 1024
)

// OpenAIClient talks to the OpenAI Chat Completions API.
type OpenAIClient struct {
	apiKey       string
	model        string
	apiURL       string
	providerName string
}

// NewOpenAIClient builds a client, resolving an optional base URL override.
func NewOpenAIClient(apiKey, model, baseURL string) *OpenAIClient {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{
		apiKey:       apiKey,
		model:        model,
		apiURL:       resolveEndpoint(baseURL, openAIDefaultAPIURL, "/v1/chat/completions", "/chat/completions"),
		providerName: fmt.Sprintf("OpenAI (%s)", model),
	}
}

// ProviderName returns a human-readable provider/model label.
func (c *OpenAIClient) ProviderName() string { return c.providerName }

// SendMessage sends a user turn with history and returns the assistant text.
func (c *OpenAIClient) SendMessage(ctx context.Context, content string, history []ChatMessage, systemPrompt string) (string, error) {
	messages := []map[string]string{{"role": "system", "content": systemPrompt}}
	messages = append(messages, wireHistory(history)...)
	messages = append(messages, map[string]string{"role": "user", "content": content})

	body := map[string]any{
		"model":      c.model,
		"max_tokens": openAIMaxTokens,
		"messages":   messages,
	}
	data, err := postJSON(ctx, c.apiURL, map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}, body)
	if err != nil {
		return "", err
	}
	return openAIExtractText(data)
}

func openAIExtractText(data []byte) (string, error) {
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", decodingError("Unexpected OpenAI response shape")
	}
	return parsed.Choices[0].Message.Content, nil
}
