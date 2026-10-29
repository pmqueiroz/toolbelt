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

func TestOpenAICompatibleAskSendsTheSchemaAndReadsTheContent(t *testing.T) {
	var got chatRequest
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"blocks\":[]}"}}]}`))
	}))
	defer server.Close()

	client, err := New(Config{Provider: "openai", Model: "m1", APIKey: "sk-1", BaseURL: server.URL + "/v1/"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	content, err := client.Ask(context.Background(), Question{Text: "hi", OS: "linux"})
	if err != nil || content != `{"blocks":[]}` {
		t.Fatalf("Ask = %q, %v", content, err)
	}
	if auth != "Bearer sk-1" {
		t.Errorf("auth = %q", auth)
	}
	if got.Model != "m1" || len(got.Messages) != 2 || got.Messages[1].Content != "hi" {
		t.Errorf("request = %#v", got)
	}
	if got.ResponseFormat["type"] != "json_schema" {
		t.Errorf("response_format = %#v", got.ResponseFormat)
	}
}

func TestOpenAICompatibleOmitsAuthWithoutKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Error("authorization sent without a key")
		}
		w.Write([]byte(`{"data":[{"id":"llama3"}]}`))
	}))
	defer server.Close()

	client, _ := New(Config{Provider: "ollama", BaseURL: server.URL}, server.Client())
	if _, err := client.Models(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAICompatibleModelsAreSortedAndUnprefixed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"models/b"},{"id":"a"}]}`))
	}))
	defer server.Close()

	client, _ := New(Config{Provider: "gemini", BaseURL: server.URL}, server.Client())
	models, err := client.Models(context.Background())
	if err != nil || !reflect.DeepEqual(models, []string{"a", "b"}) {
		t.Fatalf("Models = %v, %v", models, err)
	}
}

func TestOpenAICompatibleReportsHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, _ := New(Config{Provider: "openai", BaseURL: server.URL}, server.Client())
	_, err := client.Ask(context.Background(), Question{Text: "hi"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized || apiErr.Body != "bad key" {
		t.Fatalf("Ask err = %v", err)
	}
}

func TestOpenAICompatibleRefusalIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"","refusal":"no"}}]}`))
	}))
	defer server.Close()

	client, _ := New(Config{Provider: "openai", BaseURL: server.URL}, server.Client())
	if _, err := client.Ask(context.Background(), Question{Text: "hi"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("Ask err = %v; want ErrRefused", err)
	}
}

func TestNewRejectsUnknownProviders(t *testing.T) {
	if _, err := New(Config{Provider: "nope"}, nil); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("New err = %v", err)
	}
}
