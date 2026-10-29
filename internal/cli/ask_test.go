package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/ai"
	"github.com/pmqueiroz/toolbelt/internal/vault"
)

type fakeAI struct {
	answer string
	models []string
	asked  ai.Question
	config ai.Config
}

func (f *fakeAI) Ask(_ context.Context, q ai.Question) (string, error) {
	f.asked = q
	return f.answer, nil
}

func (f *fakeAI) Models(context.Context) ([]string, error) {
	return f.models, nil
}

func testAskDeps(t *testing.T, fake *fakeAI) askDeps {
	return askDeps{
		load: testSecretEnv(t),
		newClient: func(cfg ai.Config, _ secretEnv) (ai.Client, error) {
			fake.config = cfg
			return fake, nil
		},
	}
}

func runAsk(deps askDeps, stdin string, args ...string) (string, error) {
	cmd := newAskCmdWith(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestAskWithoutConnectionFails(t *testing.T) {
	_, err := runAsk(testAskDeps(t, &fakeAI{}), "", "what", "is", "the", "moon")
	if !errors.Is(err, errNotConnected) {
		t.Fatalf("ask = %v; want errNotConnected", err)
	}
}

func TestAskConnectThenAskPrintsThePlainAnswer(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	fake := &fakeAI{
		models: []string{"m1", "m2"},
		answer: `{"blocks":[{"type":"text","text":"Earth's natural satellite."},{"type":"command","text":"note","code":"ls -la"}]}`,
	}
	deps := testAskDeps(t, fake)
	if _, err := runAsk(deps, "", "connect", "--provider", "openai", "--key", "sk-1", "--model", "m2"); err != nil {
		t.Fatal(err)
	}
	out, err := runAsk(deps, "", "what", "is", "the", "moon")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Earth's natural satellite.\nls -la\n" {
		t.Fatalf("out = %q", out)
	}
	if fake.asked.Text != "what is the moon" {
		t.Errorf("question = %q", fake.asked.Text)
	}
	if fake.config != (ai.Config{Provider: "openai", Model: "m2", APIKey: "sk-1"}) {
		t.Errorf("config = %#v", fake.config)
	}
}

func TestAskSendsPipedInputAsContext(t *testing.T) {
	fake := &fakeAI{models: []string{"llama3"}, answer: "plain words"}
	deps := testAskDeps(t, fake)
	if _, err := runAsk(deps, "", "connect", "--provider", "ollama"); err != nil {
		t.Fatal(err)
	}
	out, err := runAsk(deps, "panic: boom\n", "why")
	if err != nil {
		t.Fatal(err)
	}
	if fake.asked.Text != "why" || fake.asked.Context != "panic: boom\n" {
		t.Errorf("question = %#v", fake.asked)
	}
	if out != "plain words\n" {
		t.Errorf("out = %q", out)
	}
}

func TestAskReadsTheQuestionFromStdinWithoutArgs(t *testing.T) {
	fake := &fakeAI{models: []string{"llama3"}, answer: "x"}
	deps := testAskDeps(t, fake)
	if _, err := runAsk(deps, "", "connect", "--provider", "ollama"); err != nil {
		t.Fatal(err)
	}
	if _, err := runAsk(deps, "define moon\n"); err != nil {
		t.Fatal(err)
	}
	if fake.asked.Text != "define moon" || fake.asked.Context != "" {
		t.Errorf("question = %#v", fake.asked)
	}
}

func TestAskWithoutQuestionFails(t *testing.T) {
	if _, err := runAsk(testAskDeps(t, &fakeAI{}), ""); !errors.Is(err, errNoQuestion) {
		t.Fatalf("ask = %v; want errNoQuestion", err)
	}
}

func TestConnectKeepsSecretsInTheEncryptedStore(t *testing.T) {
	fake := &fakeAI{models: []string{"m1"}}
	deps := testAskDeps(t, fake)
	if _, err := runAsk(deps, "", "connect", "--provider", "openai", "--key", "sk-secret-key"); err != nil {
		t.Fatal(err)
	}
	o, err := loadStore(deps.load)
	if err != nil {
		t.Fatal(err)
	}
	want := vault.AIConnection{Provider: "openai", Model: "m1", APIKey: "sk-secret-key"}
	if o.store.AI == nil || *o.store.AI != want {
		t.Fatalf("stored = %#v", o.store.AI)
	}
}

func TestConnectWithoutKeyFailsWhenNotInteractive(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	_, err := runAsk(testAskDeps(t, &fakeAI{}), "", "connect", "--provider", "openai")
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("connect = %v", err)
	}
}

func TestConnectReusesTheStoredKeyAndModelOfTheSameProvider(t *testing.T) {
	previous := &vault.AIConnection{Provider: "openai", Model: "m2", APIKey: "sk-old", BaseURL: "http://proxy"}
	fake := &fakeAI{models: []string{"m1", "m2"}}
	t.Setenv("OPENAI_API_KEY", "")
	conn, err := connectAI(context.Background(), previous, askConnectFlags{}, func(cfg ai.Config) (ai.Client, error) {
		fake.config = cfg
		return fake, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if *conn != *previous {
		t.Fatalf("conn = %#v, want %#v", conn, previous)
	}
}

func TestConnectToAnotherProviderForgetsTheOldKey(t *testing.T) {
	previous := &vault.AIConnection{Provider: "openai", Model: "m2", APIKey: "sk-old"}
	fake := &fakeAI{models: []string{"claude-opus-5-5"}}
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := connectAI(context.Background(), previous, askConnectFlags{provider: "anthropic"}, func(cfg ai.Config) (ai.Client, error) {
		return fake, nil
	})
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("connect = %v; want missing key", err)
	}
}

func TestChooseAIModelPrefersFlagThenRememberedThenFallback(t *testing.T) {
	models := []string{"a", "b", "c"}
	cases := []struct{ flag, remembered, fallback, want string }{
		{"x", "b", "c", "x"},
		{"", "b", "c", "b"},
		{"", "gone", "c", "c"},
		{"", "", "", "a"},
	}
	for _, c := range cases {
		got, err := chooseAIModel(models, c.flag, c.remembered, c.fallback)
		if err != nil || got != c.want {
			t.Errorf("chooseAIModel(%q, %q, %q) = %q, %v; want %q", c.flag, c.remembered, c.fallback, got, err, c.want)
		}
	}
	if _, err := chooseAIModel(nil, "", "", ""); !errors.Is(err, errNoModels) {
		t.Errorf("empty list err = %v", err)
	}
}
