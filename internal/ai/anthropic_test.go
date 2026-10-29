package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func fakeAnthropic(t *testing.T, message string, seen *map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-ant" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/messages":
			if seen != nil {
				json.NewDecoder(r.Body).Decode(seen)
			}
			w.Write([]byte(message))
		case "/v1/models":
			w.Write([]byte(`{"data":[{"id":"claude-haiku-5-5","type":"model"},{"id":"claude-opus-5-5","type":"model"}],"has_more":false}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
}

func TestAnthropicAskRequestsStructuredOutputAtLowEffort(t *testing.T) {
	var body map[string]any
	server := fakeAnthropic(t, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"end_turn","content":[{"type":"text","text":"{\"blocks\":[]}"}],"usage":{"input_tokens":1,"output_tokens":1}}`, &body)
	defer server.Close()

	client, _ := New(Config{Provider: "anthropic", Model: "claude-opus-5-5", APIKey: "sk-ant", BaseURL: server.URL}, server.Client())
	content, err := client.Ask(context.Background(), Question{Text: "hi"})
	if err != nil || content != `{"blocks":[]}` {
		t.Fatalf("Ask = %q, %v", content, err)
	}
	config := body["output_config"].(map[string]any)
	if config["effort"] != "low" || config["format"].(map[string]any)["type"] != "json_schema" {
		t.Errorf("output_config = %#v", config)
	}
	if body["model"] != "claude-opus-5-5" {
		t.Errorf("model = %v", body["model"])
	}
}

func TestAnthropicRefusalIsAnError(t *testing.T) {
	server := fakeAnthropic(t, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"refusal","content":[],"usage":{"input_tokens":1,"output_tokens":1}}`, nil)
	defer server.Close()

	client, _ := New(Config{Provider: "anthropic", Model: "claude-opus-5-5", APIKey: "sk-ant", BaseURL: server.URL}, server.Client())
	if _, err := client.Ask(context.Background(), Question{Text: "hi"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("Ask err = %v; want ErrRefused", err)
	}
}

func TestAnthropicModelsPutTheDefaultFirst(t *testing.T) {
	server := fakeAnthropic(t, "", nil)
	defer server.Close()

	client, _ := New(Config{Provider: "anthropic", APIKey: "sk-ant", BaseURL: server.URL}, server.Client())
	models, err := client.Models(context.Background())
	if err != nil || !reflect.DeepEqual(models, []string{"claude-opus-5-5", "claude-haiku-5-5"}) {
		t.Fatalf("Models = %v, %v", models, err)
	}
}
