package session

import (
	"context"
	"strings"

	"github.com/syriku/aisdk/api"
	"github.com/syriku/aisdk/request"
)

// Request defines the interface for an active translation session.
// It allows consumers to add text to be translated and retrieve the result.
type Request interface {
	// Translate triggers the translation of the provided text.
	// For continuous requests, this adds new text to the context.
	Translate(ctx context.Context, chatApi api.ChatCompletionsApi, text string) Request

	// TranslateAsync triggers the translation of the provided text asynchronously.
	// It pushes the translation stream and any errors to the provided channels.
	TranslateAsync(ctx context.Context, chatApi api.ChatCompletionsApi, text string, response chan<- string, errChan chan<- error) Request

	// Text returns the resulting translated text.
	Text() string

	// Error returns any error that occurred during translation.
	Error() error
}

// oneTimeRequest represents a single translation task without persistent history.
type oneTimeRequest struct {
	translator    request.Translator
	glossaries    []request.GlossaryEntry
	recentHistory []string

	text string
	err  error
}

// NewOneTimeRequest creates a new Request that performs a one-time translation.
// It uses the provided translator settings, and glossary with optional recent translation history context.
func NewOneTimeRequest(translator request.Translator, glossaries []request.GlossaryEntry, recentHistory []string) Request {
	return &oneTimeRequest{
		translator:    translator,
		glossaries:    glossaries,
		recentHistory: recentHistory,
	}
}

// TranslateAsync performs the HTTP request to the AI provider to translate the text asynchronously.
func (r *oneTimeRequest) TranslateAsync(ctx context.Context, chatApi api.ChatCompletionsApi, text string, response chan<- string, errChan chan<- error) Request {
	// 1. Build context and request data
	reqCtx := request.TranslationContext{
		Glossary:      r.glossaries,
		RecentHistory: r.recentHistory,
	}
	reqData := request.TranslateRequest{
		SourceText: text,
		Context:    reqCtx,
	}

	systemPrompt := r.translator.GenerateSystemPrompt(r.glossaries)
	userPrompt := r.translator.GenerateUserPrompt(reqData)

	chatApi.SetSystemPrompt(systemPrompt)
	chatApi.SetUserPrompt(userPrompt)

	go chatApi.ChatCompletions(ctx, response, errChan)

	return r
}

// Translate performs the HTTP request to the AI provider to translate the text synchronously.
func (r *oneTimeRequest) Translate(ctx context.Context, chatApi api.ChatCompletionsApi, text string) Request {
	responseCh := make(chan string)
	errCh := make(chan error)

	r.TranslateAsync(ctx, chatApi, text, responseCh, errCh)

	var result strings.Builder
	var finalErr error

	for responseCh != nil || errCh != nil {
		select {
		case text, ok := <-responseCh:
			if !ok {
				responseCh = nil
			} else {
				result.WriteString(text)
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
			} else {
				if finalErr == nil {
					finalErr = err
				}
			}
		}
	}

	r.text = result.String()
	r.err = finalErr

	return r
}

func (r *oneTimeRequest) Text() string {
	return r.text
}

func (r *oneTimeRequest) Error() error {
	return r.err
}
