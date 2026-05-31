package api

import (
	"context"
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
