package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ModelsApi interface {
	GetModels(ctx context.Context) ([]string, error)
}

type ChatCompletionsApi interface {
	SetSystemPrompt(systemPrompt string)
	SetUserPrompt(userPrompt string)
	SetHistory(history []struct {
		Type    CharacterType
		Content string
	})

	ChatCompletions(ctx context.Context, response chan<- string, err chan<- error)
}

type openAiModels struct {
	config UserConfig
}

type openAiModelResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (m *openAiModels) GetModels(ctx context.Context) ([]string, error) {
	baseUrl := m.config.Provider
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseUrl+"models", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+m.config.ApiKey)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get models: status %d", resp.StatusCode)
	}

	var result openAiModelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var models []string
	for _, model := range result.Data {
		models = append(models, model.ID)
	}

	return models, nil
}

func newOpenAiModels(config UserConfig) ModelsApi {
	return &openAiModels{config: config}
}

type openAiChatCompletions struct {
	config       UserConfig
	model        string
	systemPrompt string
	userPrompt   string
	history      []struct {
		Type    CharacterType
		Content string
	}
}

func newOpenAiChatCompletions(config UserConfig, model string) ChatCompletionsApi {
	return &openAiChatCompletions{
		config: config,
		model:  model,
	}
}

func (c *openAiChatCompletions) SetSystemPrompt(systemPrompt string) {
	c.systemPrompt = systemPrompt
}

func (c *openAiChatCompletions) SetUserPrompt(userPrompt string) {
	c.userPrompt = userPrompt
}

func (c *openAiChatCompletions) SetHistory(history []struct {
	Type    CharacterType
	Content string
}) {
	c.history = history
}

type openAiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAiChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAiMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type openAiChatResponseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (c *openAiChatCompletions) ChatCompletions(ctx context.Context, response chan<- string, errChan chan<- error) {
	defer close(response)
	defer close(errChan)

	var messages []openAiMessage
	if c.systemPrompt != "" {
		messages = append(messages, openAiMessage{Role: "system", Content: c.systemPrompt})
	}

	for _, h := range c.history {
		role := "user"
		if h.Type == CHARACTER_ASSISTANT {
			role = "assistant"
		}
		messages = append(messages, openAiMessage{Role: role, Content: h.Content})
	}

	if c.userPrompt != "" {
		messages = append(messages, openAiMessage{Role: "user", Content: c.userPrompt})
	}

	reqBody := openAiChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		errChan <- err
		return
	}

	baseUrl := c.config.Provider
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl+"chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		errChan <- err
		return
	}

	req.Header.Set("Authorization", "Bearer "+c.config.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		errChan <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("failed to get chat completions: status %d", resp.StatusCode)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAiChatResponseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			errChan <- fmt.Errorf("failed to decode chunk: %w", err)
			return
		}

		if len(chunk.Choices) > 0 {
			if content := chunk.Choices[0].Delta.Content; content != "" {
				response <- content
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
	}
}

type anthropicModels struct {
	config UserConfig
}

type anthropicModelResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (m *anthropicModels) GetModels(ctx context.Context) ([]string, error) {
	officialUrl := "https://api.anthropic.com/v1/models"

	req, err := http.NewRequestWithContext(ctx, "GET", officialUrl, nil)
	if err != nil {
		return []string{"claude-opus-4.7", "claude-sonnet-4.7", "claude-haiku-4.7"}, nil
	}

	req.Header.Set("x-api-key", m.config.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return []string{"claude-opus-4.7", "claude-sonnet-4.7", "claude-haiku-4.7"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{"claude-opus-4.7", "claude-sonnet-4.7", "claude-haiku-4.7"}, nil
	}

	var result anthropicModelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []string{"claude-opus-4.7", "claude-sonnet-4.7", "claude-haiku-4.7"}, nil
	}

	var models []string
	for _, model := range result.Data {
		models = append(models, model.ID)
	}

	if len(models) == 0 {
		return []string{"claude-opus-4.7", "claude-sonnet-4.7", "claude-haiku-4.7"}, nil
	}

	return models, nil
}

func newAnthropicModels(config UserConfig) ModelsApi {
	return &anthropicModels{config: config}
}

type anthropicChatCompletions struct {
	config       UserConfig
	model        string
	systemPrompt string
	userPrompt   string
	history      []struct {
		Type    CharacterType
		Content string
	}
}

func newAnthropicChatCompletions(config UserConfig, model string) ChatCompletionsApi {
	return &anthropicChatCompletions{
		config: config,
		model:  model,
	}
}

func (c *anthropicChatCompletions) SetSystemPrompt(systemPrompt string) {
	c.systemPrompt = systemPrompt
}

func (c *anthropicChatCompletions) SetUserPrompt(userPrompt string) {
	c.userPrompt = userPrompt
}

func (c *anthropicChatCompletions) SetHistory(history []struct {
	Type    CharacterType
	Content string
}) {
	c.history = history
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicChatRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
}

type anthropicChatResponseChunk struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *anthropicChatCompletions) ChatCompletions(ctx context.Context, response chan<- string, errChan chan<- error) {
	defer close(response)
	defer close(errChan)

	var messages []anthropicMessage
	for _, h := range c.history {
		role := "user"
		if h.Type == CHARACTER_ASSISTANT {
			role = "assistant"
		}
		messages = append(messages, anthropicMessage{Role: role, Content: h.Content})
	}

	if c.userPrompt != "" {
		messages = append(messages, anthropicMessage{Role: "user", Content: c.userPrompt})
	}

	if len(messages) == 0 {
		errChan <- fmt.Errorf("at least one message is required")
		return
	}

	reqBody := anthropicChatRequest{
		Model:     c.model,
		Messages:  messages,
		System:    c.systemPrompt,
		MaxTokens: 4096,
		Stream:    true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		errChan <- err
		return
	}

	baseUrl := c.config.Provider
	if baseUrl == "" {
		baseUrl = "https://api.anthropic.com/v1"
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl+"messages", bytes.NewReader(bodyBytes))
	if err != nil {
		errChan <- err
		return
	}

	req.Header.Set("x-api-key", c.config.ApiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		errChan <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("failed to get chat completions: status %d", resp.StatusCode)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var chunk anthropicChatResponseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			errChan <- fmt.Errorf("failed to decode chunk: %w", err)
			return
		}

		if chunk.Type == "error" && chunk.Error != nil {
			errChan <- fmt.Errorf("api error: [%s] %s", chunk.Error.Type, chunk.Error.Message)
			return
		}

		if chunk.Type == "content_block_delta" && chunk.Delta.Type == "text_delta" {
			if chunk.Delta.Text != "" {
				response <- chunk.Delta.Text
			}
		}

		if chunk.Type == "message_stop" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- err
	}
}
