package api

// ApiFactory defines the interface for an AI provider factory.
// Note that the factory does not directly return the final execution results.
// Instead, it constructs and returns the necessary requests tailored for different
// providers to achieve a specific goal. The caller must then use an HTTP client
// to send these constructed requests to the respective provider's server to
// obtain the actual results.
type ApiFactory interface {
	Models() ModelsApi
	ChatCompletions(model string) ChatCompletionsApi
}

type openAiFactory struct {
	userConfig UserConfig
}

func newOpenAiFactory(userConfig UserConfig) ApiFactory {
	return &openAiFactory{
		userConfig: userConfig,
	}
}

func (o *openAiFactory) Models() ModelsApi {
	return newOpenAiModels(o.userConfig)
}

func (o *openAiFactory) ChatCompletions(model string) ChatCompletionsApi {
	return newOpenAiChatCompletions(o.userConfig, model)
}

type anthropicFactory struct {
	userConfig UserConfig
}

func newAnthropicFactory(userConfig UserConfig) ApiFactory {
	return &anthropicFactory{
		userConfig: userConfig,
	}
}

func (a *anthropicFactory) Models() ModelsApi {
	return newAnthropicModels(a.userConfig)
}

func (a *anthropicFactory) ChatCompletions(model string) ChatCompletionsApi {
	return newAnthropicChatCompletions(a.userConfig, model)
}

// NewFactory creates and returns a new ApiFactory instance based on the Type in UserConfig.
func NewFactory(userConfig UserConfig) ApiFactory {
	switch userConfig.Type {
	case API_TYPE_OPEN_AI:
		return newOpenAiFactory(userConfig)
	case API_TYPE_CLAUDE:
		return newAnthropicFactory(userConfig)
	default:
		return nil
	}
}
