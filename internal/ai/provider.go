package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

type Preset struct {
	Name         string
	Title        string
	BaseURL      string
	KeyEnv       string
	DefaultModel string
	anthropic    bool
}

func (p Preset) NeedsKey() bool {
	return p.KeyEnv != ""
}

var presets = []Preset{
	{Name: "anthropic", Title: "Anthropic (Claude)", KeyEnv: "ANTHROPIC_API_KEY", DefaultModel: "claude-opus-5-5", anthropic: true},
	{Name: "openai", Title: "OpenAI", BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY"},
	{Name: "gemini", Title: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", KeyEnv: "GEMINI_API_KEY"},
	{Name: "openrouter", Title: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", KeyEnv: "OPENROUTER_API_KEY"},
	{Name: "ollama", Title: "Ollama (local)", BaseURL: "http://localhost:11434/v1"},
}

var ErrUnknownProvider = errors.New("unknown provider")

func Presets() []Preset {
	return presets
}

func FindPreset(name string) (Preset, error) {
	for _, p := range presets {
		if p.Name == name {
			return p, nil
		}
	}
	return Preset{}, fmt.Errorf("%w %q", ErrUnknownProvider, name)
}

type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

type Client interface {
	Ask(ctx context.Context, q Question) (string, error)
	Models(ctx context.Context) ([]string, error)
}

func New(cfg Config, httpClient *http.Client) (Client, error) {
	preset, err := FindPreset(cfg.Provider)
	if err != nil {
		return nil, err
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = preset.BaseURL
	}
	if preset.anthropic {
		return newAnthropic(cfg.APIKey, baseURL, cfg.Model, httpClient), nil
	}
	return &openAICompatible{baseURL: baseURL, key: cfg.APIKey, model: cfg.Model, http: httpClient}, nil
}

func sortedModels(models []string, preferred string) []string {
	sort.Strings(models)
	for i, m := range models {
		if m == preferred {
			return append([]string{m}, append(models[:i:i], models[i+1:]...)...)
		}
	}
	return models
}
