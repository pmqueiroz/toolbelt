package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type openAICompatible struct {
	baseURL string
	key     string
	model   string
	http    *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string         `json:"model"`
	Messages       []chatMessage  `json:"messages"`
	ResponseFormat map[string]any `json:"response_format"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
}

type modelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("provider answered %d: %s", e.Status, e.Body)
}

func (c *openAICompatible) Ask(ctx context.Context, q Question) (string, error) {
	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: System(q)},
			{Role: "user", Content: User(q)},
		},
		ResponseFormat: map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "answer",
				"strict": true,
				"schema": Schema(),
			},
		},
	}
	var resp chatResponse
	if err := c.do(ctx, http.MethodPost, "chat/completions", body, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("provider returned no answer")
	}
	message := resp.Choices[0].Message
	if message.Refusal != "" {
		return "", fmt.Errorf("%w: %s", ErrRefused, message.Refusal)
	}
	return message.Content, nil
}

func (c *openAICompatible) Models(ctx context.Context) ([]string, error) {
	var list modelList
	if err := c.do(ctx, http.MethodGet, "models", nil, &list); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(list.Data))
	for _, m := range list.Data {
		models = append(models, strings.TrimPrefix(m.ID, "models/"))
	}
	return sortedModels(models, ""), nil
}

func (c *openAICompatible) do(ctx context.Context, method, path string, in, out any) error {
	var payload io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.baseURL, "/")+"/"+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	httpClient := c.http
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return &APIError{Status: resp.StatusCode, Body: strings.TrimSpace(string(raw))}
	}
	return json.Unmarshal(raw, out)
}
