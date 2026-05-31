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
