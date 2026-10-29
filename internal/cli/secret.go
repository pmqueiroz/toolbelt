package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/pmqueiroz/toolbelt/internal/bitwarden"
	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/secret"
	"github.com/pmqueiroz/toolbelt/internal/ui"
	"github.com/pmqueiroz/toolbelt/internal/vault"
	"github.com/spf13/cobra"
)

const (
	tokenEnv      = "BWS_ACCESS_TOKEN"
	defaultRegion = "us"
)

var (
	errNoValue    = errors.New("no value given, pass it as an argument or pipe it through stdin")
	errNoToken    = errors.New("no access token, pass --token or set " + tokenEnv)
	errNoProjects = errors.New("the machine account sees no project, grant it access to one in the bitwarden web app")
)

type secretEnv struct {
	paths     vault.Paths
	keyring   vault.Keyring
	http      *http.Client
	endpoints func(region string) (bitwarden.Endpoints, error)
}

func defaultSecretEnv() (secretEnv, error) {
	paths, err := vault.DefaultPaths()
	if err != nil {
		return secretEnv{}, err
	}
	return secretEnv{
		paths:     paths,
		keyring:   vault.OSKeyring{},
		http:      &http.Client{Timeout: 30 * time.Second},
		endpoints: bitwarden.Region,
	}, nil
}

func newSecretCmd() *cobra.Command {
	return newSecretCmdWith(defaultSecretEnv)
}

func newSecretCmdWith(load func() (secretEnv, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Encrypted key/value secrets, mirrored from Bitwarden Secrets Manager once connected",
	}
	cmd.AddCommand(
		newSecretSetCmd(load),
		newSecretGetCmd(load),
		newSecretListCmd(load),
		newSecretConnectCmd(load),
	)
	return cmd
}

type openStore struct {
	env   secretEnv
	key   []byte
	store *vault.Store
}

func loadStore(load func() (secretEnv, error)) (*openStore, error) {
	env, err := load()
	if err != nil {
		return nil, err
	}
	key, err := vault.LoadDataKey(env.keyring, env.paths)
	if err != nil {
		return nil, err
	}
	if key.Created && key.FromFile {
		home, _ := os.UserHomeDir()
		ui.Warn("os keyring unavailable, secret key kept in a file", "path", files.Pretty(env.paths.KeyFile, home))
	}
	store, err := vault.Load(env.paths.Store, key.Bytes)
	if err != nil {
		return nil, err
	}
	return &openStore{env: env, key: key.Bytes, store: store}, nil
}

func (o *openStore) save() error {
	return o.store.Save(o.env.paths.Store, o.key)
}

func (o *openStore) login(ctx context.Context, raw, region string) (*bitwarden.Session, error) {
	token, err := bitwarden.ParseAccessToken(raw)
	if err != nil {
		return nil, err
	}
	endpoints, err := o.env.endpoints(region)
	if err != nil {
		return nil, err
	}
	var session *bitwarden.Session
	err = ui.Spin("logging in to bitwarden", func(ctx context.Context) error {
		session, err = bitwarden.Login(ctx, endpoints, o.env.http, token)
		return err
	})
	return session, err
}

func newSecretSetCmd(load func() (secretEnv, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY [VALUE]",
		Short: "Store a secret; the value is read from stdin when omitted",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := secretValue(cmd.InOrStdin(), args)
			if err != nil {
				return err
			}
			o, err := loadStore(load)
			if err != nil {
				return err
			}
			key := strings.TrimSpace(args[0])
			entry := vault.Entry{Value: value}
			if current, err := o.store.Get(key); err == nil {
				entry.ID = current.ID
			}
			if o.store.Connected() {
				if entry, err = o.push(cmd.Context(), key, entry); err != nil {
					return err
				}
			}
			if err := o.store.Set(key, entry); err != nil {
				return err
			}
			return o.save()
		},
	}
}

func (o *openStore) push(ctx context.Context, key string, entry vault.Entry) (vault.Entry, error) {
	conn := o.store.Connection
	session, err := o.login(ctx, conn.Token, conn.Region)
	if err != nil {
		return entry, err
	}
	var saved bitwarden.Secret
	err = ui.Spin("saving "+key+" to "+conn.ProjectName, func(ctx context.Context) error {
		saved, err = upsert(ctx, session, conn.ProjectID, key, entry)
		return err
	})
	if err != nil {
		return entry, err
	}
	return vault.Entry{Value: entry.Value, ID: saved.ID}, nil
}

func upsert(ctx context.Context, session *bitwarden.Session, projectID, key string, entry vault.Entry) (bitwarden.Secret, error) {
	if entry.ID != "" {
		saved, err := session.Update(ctx, projectID, bitwarden.Secret{ID: entry.ID, Key: key, Value: entry.Value})
		var apiErr *bitwarden.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
			return saved, err
		}
	}
	return session.Create(ctx, projectID, key, entry.Value)
}

func secretValue(stdin io.Reader, args []string) (string, error) {
	if len(args) == 2 {
		return args[1], nil
	}
	if ui.Interactive() {
		var value string
		err := huh.NewInput().Title(args[0]).EchoMode(huh.EchoModePassword).Value(&value).Run()
		if err != nil {
			return "", err
		}
		return value, nil
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(raw), "\n"), "\r")
	if value == "" {
		return "", errNoValue
	}
	return value, nil
}

func newSecretGetCmd(load func() (secretEnv, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Print a secret from the local store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := loadStore(load)
			if err != nil {
				return err
			}
			entry, err := o.store.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), entry.Value)
			return nil
		},
	}
}

func newSecretListCmd(load func() (secretEnv, error)) *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every secret, masked unless --reveal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o, err := loadStore(load)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, key := range o.store.Keys() {
				value := o.store.Entries[key].Value
				if !reveal {
					value = secret.Redact(value)
				}
				fmt.Fprintf(out, "%s\t%s\n", key, value)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "print values unmasked")
	return cmd
}

type connectFlags struct {
	token   string
	project string
	region  string
}

func newSecretConnectCmd(load func() (secretEnv, error)) *cobra.Command {
	var flags connectFlags
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Link the store to a Bitwarden Secrets Manager project and mirror it locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o, err := loadStore(load)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("region") && o.store.Connected() {
				flags.region = o.store.Connection.Region
			}
			return o.connect(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.token, "token", "", "machine account access token (default $"+tokenEnv+")")
	cmd.Flags().StringVar(&flags.project, "project", "", "project name or id")
	cmd.Flags().StringVar(&flags.region, "region", defaultRegion, "bitwarden cloud region: us or eu")
	return cmd
}

func (o *openStore) connect(ctx context.Context, flags connectFlags) error {
	previous := o.store.Connection
	token, err := accessToken(flags.token, previous)
	if err != nil {
		return err
	}
	session, err := o.login(ctx, token, flags.region)
	if err != nil {
		return err
	}

	var projects []bitwarden.Project
	err = ui.Spin("listing projects", func(ctx context.Context) error {
		projects, err = session.Projects(ctx)
		return err
	})
	if err != nil {
		return err
	}
	remembered := ""
	if previous != nil {
		remembered = previous.ProjectID
	}
	project, err := chooseProject(projects, flags.project, remembered)
	if err != nil {
		return err
	}

	var secrets []bitwarden.Secret
	err = ui.Spin("pulling "+project.Name, func(ctx context.Context) error {
		secrets, err = session.Secrets(ctx, project.ID)
		return err
	})
	if err != nil {
		return err
	}
	remote, duplicates := remoteEntries(secrets)
	for _, key := range duplicates {
		ui.Warn("key used by more than one secret in the project, keeping the first", "key", key)
	}

	local := o.store.Entries
	if previous == nil || previous.ProjectID != project.ID {
		local = forgetRemoteIDs(local)
	}
	merged, upload := vault.Merge(local, remote)
	o.store.Entries = merged
	o.store.Connection = &vault.Connection{
		Token:          token,
		Region:         flags.region,
		OrganizationID: session.OrganizationID,
		ProjectID:      project.ID,
		ProjectName:    project.Name,
	}

	if len(upload) > 0 {
		o.uploadLocalOnly(ctx, session, project, upload)
	}
	if err := o.save(); err != nil {
		return err
	}
	ui.Info("connected", "project", project.Name, "secrets", len(o.store.Entries))
	return nil
}

func (o *openStore) uploadLocalOnly(ctx context.Context, session *bitwarden.Session, project bitwarden.Project, keys []string) {
	question := fmt.Sprintf("upload %d local-only secret(s) to %s? (%s)", len(keys), project.Name, strings.Join(keys, ", "))
	if !ui.Interactive() || !ui.Confirm(question) {
		ui.Warn("local-only secrets kept on this machine only", "keys", strings.Join(keys, ","))
		return
	}
	for _, key := range keys {
		entry := o.store.Entries[key]
		var saved bitwarden.Secret
		err := ui.Spin("uploading "+key, func(ctx context.Context) error {
			var err error
			saved, err = session.Create(ctx, project.ID, key, entry.Value)
			return err
		})
		if err != nil {
			ui.Error("upload failed, kept locally", "key", key, "err", err)
			continue
		}
		o.store.Entries[key] = vault.Entry{Value: entry.Value, ID: saved.ID}
	}
}

func accessToken(flag string, previous *vault.Connection) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env := os.Getenv(tokenEnv); env != "" {
		return env, nil
	}
	if previous != nil && previous.Token != "" {
		return previous.Token, nil
	}
	if !ui.Interactive() {
		return "", errNoToken
	}
	var token string
	err := huh.NewInput().
		Title("bitwarden access token").
		Description("machine account access token from the secrets manager web app").
		EchoMode(huh.EchoModePassword).
		Value(&token).
		Run()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", errNoToken
	}
	return token, nil
}

func chooseProject(projects []bitwarden.Project, flag, remembered string) (bitwarden.Project, error) {
	if len(projects) == 0 {
		return bitwarden.Project{}, errNoProjects
	}
	if flag != "" {
		if p, ok := findProject(projects, flag); ok {
			return p, nil
		}
		return bitwarden.Project{}, fmt.Errorf("project %q not found", flag)
	}
	if p, ok := findProject(projects, remembered); ok {
		return p, nil
	}
	if len(projects) == 1 {
		return projects[0], nil
	}
	if !ui.Interactive() {
		return bitwarden.Project{}, errors.New("more than one project, pick one with --project")
	}
	options := make([]huh.Option[int], len(projects))
	for i, p := range projects {
		options[i] = huh.NewOption(p.Name, i)
	}
	var chosen int
	if err := huh.NewSelect[int]().Title("project").Options(options...).Value(&chosen).Run(); err != nil {
		return bitwarden.Project{}, err
	}
	return projects[chosen], nil
}

func findProject(projects []bitwarden.Project, idOrName string) (bitwarden.Project, bool) {
	if idOrName == "" {
		return bitwarden.Project{}, false
	}
	for _, p := range projects {
		if p.ID == idOrName || p.Name == idOrName {
			return p, true
		}
	}
	return bitwarden.Project{}, false
}

func remoteEntries(secrets []bitwarden.Secret) (map[string]vault.Entry, []string) {
	entries := make(map[string]vault.Entry, len(secrets))
	var duplicates []string
	for _, s := range secrets {
		if _, seen := entries[s.Key]; seen {
			duplicates = append(duplicates, s.Key)
			continue
		}
		entries[s.Key] = vault.Entry{Value: s.Value, ID: s.ID}
	}
	return entries, duplicates
}

func forgetRemoteIDs(entries map[string]vault.Entry) map[string]vault.Entry {
	local := make(map[string]vault.Entry, len(entries))
	for key, entry := range entries {
		local[key] = vault.Entry{Value: entry.Value}
	}
	return local
}
