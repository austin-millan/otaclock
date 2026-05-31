package core

import (
	"context"
	"fmt"
	"time"
)

// Role identifies the author of a chat message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// ChatMessage is a single turn in the conversation transcript.
type ChatMessage struct {
	Role      Role
	Content   string
	Timestamp time.Time
}

// NewChatMessage stamps a message with the current time.
func NewChatMessage(role Role, content string) ChatMessage {
	return ChatMessage{Role: role, Content: content, Timestamp: time.Now()}
}

// LLMError describes a failure talking to a provider.
type LLMError struct {
	Kind       string
	StatusCode int
	Message    string
}

func (e *LLMError) Error() string {
	switch e.Kind {
	case "missing_api_key":
		return fmt.Sprintf("Missing API key for %s. Set the appropriate environment variable.", e.Message)
	case "http":
		return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	case "decoding":
		return fmt.Sprintf("Failed to decode response: %s", e.Message)
	default:
		return fmt.Sprintf("Network error: %s", e.Message)
	}
}

func httpError(status int, body string) *LLMError {
	return &LLMError{Kind: "http", StatusCode: status, Message: body}
}

func decodingError(msg string) *LLMError { return &LLMError{Kind: "decoding", Message: msg} }

func networkError(msg string) *LLMError { return &LLMError{Kind: "network", Message: msg} }

// LLMClient sends chat messages to a provider, including history for context.
type LLMClient interface {
	ProviderName() string
	SendMessage(ctx context.Context, content string, history []ChatMessage, systemPrompt string) (string, error)
}

// SendMessageDefaultPrompt is a convenience that sends with the Otacon persona.
func SendMessageDefaultPrompt(ctx context.Context, client LLMClient, content string, history []ChatMessage) (string, error) {
	return client.SendMessage(ctx, content, history, SystemPrompt)
}
