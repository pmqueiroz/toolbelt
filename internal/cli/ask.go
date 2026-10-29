package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/pmqueiroz/toolbelt/internal/ai"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/pmqueiroz/toolbelt/internal/vault"
	"github.com/spf13/cobra"
)

var (
	errNoQuestion   = errors.New("no question, pass it as arguments or pipe it through stdin")
	errNotConnected = errors.New("no ai provider connected, run tb ask connect")
	errNoProvider   = errors.New("no provider, pass --provider")
	errNoModels     = errors.New("the provider lists no model, pass --model")
)

type askDeps struct {
	load      func() (secretEnv, error)
	newClient func(ai.Config, secretEnv) (ai.Client, error)
}

func newAskCmd() *cobra.Command {
	return newAskCmdWith(askDeps{
		load: defaultSecretEnv,
		newClient: func(cfg ai.Config, env secretEnv) (ai.Client, error) {
			return ai.New(cfg, env.http)
		},
	})
}

func newAskCmdWith(deps askDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [QUESTION...]",
		Short: "Ask an ai a quick question and get a short answer, or a command to run",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := askQuestion(cmd.InOrStdin(), args)
			if err != nil {
				return err
			}
			if len(q.Context) > ai.MaxContextBytes {
				ui.Warn("piped input is long, only its end is sent", "kept", ai.MaxContextBytes)
			}
			o, err := loadStore(deps.load)
			if err != nil {
				return err
			}
			if o.store.AI == nil {
				return errNotConnected
			}
			client, err := deps.newClient(aiConfig(o.store.AI), o.env)
			if err != nil {
				return err
			}
			var raw string
			err = ui.Spin("thinking", func(ctx context.Context) error {
				raw, err = client.Ask(ctx, q)
				return err
			})
			if err != nil {
				return err
			}
			answer := ai.Parse(raw)
			out := cmd.OutOrStdout()
			width, styled := terminalWidth(out)
			if !styled {
				fmt.Fprint(out, renderAnswerPlain(answer))
				return nil
			}
			fmt.Fprintln(out, renderAnswer(answer, min(width, maxAnswerWidth)))
			if commands := answer.Commands(); len(commands) > 0 && ui.Interactive() {
				return offerCommands(cmd.Context(), commands)
			}
			return nil
		},
	}
	cmd.AddCommand(newAskConnectCmd(deps))
	return cmd
}

func askQuestion(stdin io.Reader, args []string) (ai.Question, error) {
	q := ai.Question{
		Text:  strings.TrimSpace(strings.Join(args, " ")),
		OS:    runtime.GOOS,
		Shell: userShell(),
	}
	if !ui.Interactive() {
		piped, err := io.ReadAll(stdin)
		if err != nil {
			return q, err
		}
		if q.Text == "" {
			q.Text = strings.TrimSpace(string(piped))
		} else {
			q.Context = string(piped)
		}
	} else if q.Text == "" {
		if err := huh.NewInput().Title("ask").Value(&q.Text).Run(); err != nil {
			return q, err
		}
		q.Text = strings.TrimSpace(q.Text)
	}
	if q.Text == "" {
		return q, errNoQuestion
	}
	return q, nil
}

func userShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return filepath.Base(os.Getenv("SHELL"))
}

func aiConfig(conn *vault.AIConnection) ai.Config {
	return ai.Config{Provider: conn.Provider, Model: conn.Model, APIKey: conn.APIKey, BaseURL: conn.BaseURL}
}

func offerCommands(ctx context.Context, commands []string) error {
	const done = -1
	options := make([]huh.Option[int], 0, len(commands)+1)
	for i, c := range commands {
		options = append(options, huh.NewOption("run  "+strings.ReplaceAll(c, "\n", " ⏎ "), i))
	}
	options = append(options, huh.NewOption("done", done))
	chosen := 0
	if err := huh.NewSelect[int]().Options(options...).Value(&chosen).Run(); err != nil || chosen == done {
		return nil
	}
	command := commands[chosen]
	if err := editCommand(&command); err != nil {
		return nil
	}
	if strings.TrimSpace(command) == "" {
		return nil
	}
	shell, flag := shellCommand()
	return quietRunner().Attached(ctx, shell, flag, command)
}

func editCommand(command *string) error {
	const title, hint = "run", "edit before running, esc to cancel"
	if strings.Contains(*command, "\n") {
		return huh.NewText().Title(title).Description(hint).Value(command).Run()
	}
	return huh.NewInput().Title(title).Description(hint).Value(command).Run()
}

type askConnectFlags struct {
	provider string
	model    string
	key      string
	baseURL  string
}

func newAskConnectCmd(deps askDeps) *cobra.Command {
	var flags askConnectFlags
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Pick the ai provider, api key and model tb ask uses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o, err := loadStore(deps.load)
			if err != nil {
				return err
			}
			conn, err := connectAI(cmd.Context(), o.store.AI, flags, func(cfg ai.Config) (ai.Client, error) {
				return deps.newClient(cfg, o.env)
			})
			if err != nil {
				return err
			}
			o.store.AI = conn
			if err := o.save(); err != nil {
				return err
			}
			ui.Info("connected", "provider", conn.Provider, "model", conn.Model)
			return nil
		},
	}
	names := make([]string, len(ai.Presets()))
	for i, p := range ai.Presets() {
		names[i] = p.Name
	}
	cmd.Flags().StringVar(&flags.provider, "provider", "", "ai provider: "+strings.Join(names, ", "))
	cmd.Flags().StringVar(&flags.model, "model", "", "model id (default: pick from the provider's list)")
	cmd.Flags().StringVar(&flags.key, "key", "", "api key (default: the provider's env var, then the stored key)")
	cmd.Flags().StringVar(&flags.baseURL, "base-url", "", "api base url, for proxies or openai-compatible servers")
	return cmd
}

func connectAI(ctx context.Context, previous *vault.AIConnection, flags askConnectFlags, newClient func(ai.Config) (ai.Client, error)) (*vault.AIConnection, error) {
	preset, err := chooseAIProvider(flags.provider, previous)
	if err != nil {
		return nil, err
	}
	same := previous != nil && previous.Provider == preset.Name
	cfg := ai.Config{Provider: preset.Name, BaseURL: flags.baseURL}
	if cfg.BaseURL == "" && same {
		cfg.BaseURL = previous.BaseURL
	}
	if preset.NeedsKey() {
		stored := ""
		if same {
			stored = previous.APIKey
		}
		if cfg.APIKey, err = aiKey(preset, flags.key, stored); err != nil {
			return nil, err
		}
	}
	client, err := newClient(cfg)
	if err != nil {
		return nil, err
	}
	var models []string
	err = ui.Spin("listing "+preset.Name+" models", func(ctx context.Context) error {
		models, err = client.Models(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	remembered := ""
	if same {
		remembered = previous.Model
	}
	if cfg.Model, err = chooseAIModel(models, flags.model, remembered, preset.DefaultModel); err != nil {
		return nil, err
	}
	return &vault.AIConnection{Provider: cfg.Provider, Model: cfg.Model, APIKey: cfg.APIKey, BaseURL: cfg.BaseURL}, nil
}

func chooseAIProvider(flag string, previous *vault.AIConnection) (ai.Preset, error) {
	if flag != "" {
		return ai.FindPreset(flag)
	}
	if !ui.Interactive() {
		if previous != nil {
			return ai.FindPreset(previous.Provider)
		}
		return ai.Preset{}, errNoProvider
	}
	options := make([]huh.Option[string], len(ai.Presets()))
	for i, p := range ai.Presets() {
		options[i] = huh.NewOption(p.Title, p.Name)
	}
	chosen := ai.Presets()[0].Name
	if previous != nil {
		chosen = previous.Provider
	}
	if err := huh.NewSelect[string]().Title("provider").Options(options...).Value(&chosen).Run(); err != nil {
		return ai.Preset{}, err
	}
	return ai.FindPreset(chosen)
}

func aiKey(preset ai.Preset, flag, stored string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env := os.Getenv(preset.KeyEnv); env != "" {
		return env, nil
	}
	if stored != "" {
		return stored, nil
	}
	missing := fmt.Errorf("no api key, pass --key or set %s", preset.KeyEnv)
	if !ui.Interactive() {
		return "", missing
	}
	var key string
	err := huh.NewInput().
		Title(preset.Title + " api key").
		EchoMode(huh.EchoModePassword).
		Value(&key).
		Run()
	if err != nil {
		return "", err
	}
	if key = strings.TrimSpace(key); key == "" {
		return "", missing
	}
	return key, nil
}

func chooseAIModel(models []string, flag, remembered, fallback string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if len(models) == 0 {
		return "", errNoModels
	}
	if !ui.Interactive() {
		for _, m := range []string{remembered, fallback} {
			if m != "" && contains(models, m) {
				return m, nil
			}
		}
		return models[0], nil
	}
	chosen := models[0]
	if remembered != "" && contains(models, remembered) {
		chosen = remembered
	}
	err := huh.NewSelect[string]().
		Title("model").
		Options(huh.NewOptions(models...)...).
		Height(12).
		Value(&chosen).
		Run()
	return chosen, err
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
