package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// sharedHTTPClient is reused across requests to pool connections.
var sharedHTTPClient = &http.Client{Timeout: 60 * time.Second}

// postJSON sends a JSON body and returns the 200-OK response bytes, mapping
// failures to *LLMError.
func postJSON(ctx context.Context, url string, headers map[string]string, body map[string]any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, networkError(err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, networkError(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	for field, value := range headers {
		req.Header.Set(field, value)
	}

	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return nil, networkError(err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, networkError(err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		payloadStr := string(data)
		if payloadStr == "" {
			payloadStr = "empty body"
		}
		return nil, httpError(resp.StatusCode, payloadStr)
	}
	return data, nil
}

// resolveEndpoint resolves a user-supplied base URL to the concrete endpoint,
// tolerating partial inputs.
//
//   - fullSuffix is appended when the base URL has no /v1 component (e.g. /v1/messages).
//   - v1Tail is appended when the base URL already ends in /v1 (e.g. /messages).
func resolveEndpoint(baseURL, fallback, fullSuffix, v1Tail string) string {
	if baseURL == "" {
		return fallback
	}
	s := strings.TrimSpace(baseURL)
	for strings.HasSuffix(s, "/") {
		s = strings.TrimSuffix(s, "/")
	}
	if strings.HasSuffix(s, fullSuffix) {
		return s
	}
	if strings.HasSuffix(s, "/v1") {
		return s + v1Tail
	}
	if strings.Contains(s, "/v1/") {
		return s
	}
	return s + fullSuffix
}

// wireHistory maps history (excluding system turns) to the role/content wire format.
func wireHistory(history []ChatMessage) []map[string]string {
	out := make([]map[string]string, 0, len(history))
	for _, m := range history {
		if m.Role == RoleSystem {
			continue
		}
		out = append(out, map[string]string{"role": string(m.Role), "content": m.Content})
	}
	return out
}
