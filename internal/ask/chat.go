/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package ask

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatCompletion performs a non-streaming chat completion.
// provider should be "ollama", "lmstudio", "openai", etc.
// baseEndpoint is the raw endpoint (e.g. http://localhost:11434), NOT the /v1 suffixed one.
// For Ollama, uses native /api/chat with think:false to disable chain-of-thought.
// For others, uses OpenAI-compatible /v1/chat/completions.
func ChatCompletion(ctx context.Context, provider, baseEndpoint, model, apiKey string, messages []Message, maxTokens int, temperature float32) (string, error) {
	baseEndpoint = strings.TrimRight(baseEndpoint, "/")

	if provider == "ollama" {
		return ollamaChatCompletion(ctx, baseEndpoint, model, messages, maxTokens, temperature)
	}
	return openaiChatCompletion(ctx, baseEndpoint, model, apiKey, messages, maxTokens, temperature)
}

// ollamaChatCompletion uses Ollama's native /api/chat endpoint where think:false works.
func ollamaChatCompletion(ctx context.Context, baseEndpoint, model string, messages []Message, maxTokens int, temperature float32) (string, error) {
	type ollamaMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type ollamaRequest struct {
		Model    string          `json:"model"`
		Messages []ollamaMessage `json:"messages"`
		Stream   bool            `json:"stream"`
		Think    bool            `json:"think"`
		Options  map[string]any  `json:"options,omitempty"`
	}

	msgs := make([]ollamaMessage, len(messages))
	for i, m := range messages {
		msgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	opts := map[string]any{}
	if maxTokens > 0 {
		opts["num_predict"] = maxTokens
	}
	if temperature > 0 {
		opts["temperature"] = temperature
	}

	reqBody := ollamaRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
		Think:    false,
		Options:  opts,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := baseEndpoint + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("parse ollama response: %w (body: %.200s)", err, string(respBody))
	}

	return ollamaResp.Message.Content, nil
}

// openaiChatCompletion uses the standard OpenAI /v1/chat/completions endpoint.
func openaiChatCompletion(ctx context.Context, baseEndpoint, model, apiKey string, messages []Message, maxTokens int, temperature float32) (string, error) {
	reqBody := chatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := baseEndpoint
	if !strings.Contains(endpoint, "/v1") {
		endpoint += "/v1"
	}
	url := endpoint + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %.200s)", err, string(respBody))
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices (body: %.500s)", string(respBody))
	}

	return chatResp.Choices[0].Message.Content, nil
}
