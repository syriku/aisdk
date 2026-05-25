package api

const (
	API_TYPE_OPEN_AI = iota
	API_TYPE_CLAUDE
)

const (
	CHARACTER_USER = iota
	CHARACTER_ASSISTANT
)

type ApiType int

type CharacterType int

type UserConfig struct {
	Provider string  `json:"provider"`
	Type     ApiType `json:"type"`
	ApiKey   string  `json:"api_key"`
}

// NewUserConfig creates and returns a new UserConfig instance with the specified provider, API type, and API key.
func NewUserConfig(provider string, apiType ApiType, apiKey string) UserConfig {
	return UserConfig{
		Provider: provider,
		Type:     apiType,
		ApiKey:   apiKey,
	}
}
