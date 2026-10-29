package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

const anthropicMaxTokens = 4096

var ErrRefused = errors.New("the model declined to answer")

type anthropicClient struct {
	client anthropic.Client
	model  string
}

func newAnthropic(key, baseURL, model string, httpClient *http.Client) *anthropicClient {
	opts := []option.RequestOption{option.WithAPIKey(key)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if httpClient != nil {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}
	return &anthropicClient{client: anthropic.NewClient(opts...), model: model}
}

func (c *anthropicClient) Ask(ctx context.Context, q Question) (string, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: anthropicMaxTokens,
		System:    []anthropic.TextBlockParam{{Text: System(q)}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(User(q))),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortLow,
			Format: anthropic.JSONOutputFormatParam{Schema: Schema()},
		},
	})
	if err != nil {
		return "", err
	}
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", ErrRefused
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	return text.String(), nil
}

func (c *anthropicClient) Models(ctx context.Context) ([]string, error) {
	pager := c.client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	var models []string
	for pager.Next() {
		models = append(models, pager.Current().ID)
	}
	if err := pager.Err(); err != nil {
		return nil, err
	}
	return sortedModels(models, presets[0].DefaultModel), nil
}
