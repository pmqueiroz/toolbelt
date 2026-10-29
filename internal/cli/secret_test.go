package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pmqueiroz/toolbelt/internal/bitwarden"
	"github.com/pmqueiroz/toolbelt/internal/vault"
)

type memoryKeyring map[string]string

func (m memoryKeyring) Get(service, user string) (string, error) {
	secret, ok := m[service+"/"+user]
	if !ok {
		return "", vault.ErrKeyringItemMissing
	}
	return secret, nil
}

func (m memoryKeyring) Set(service, user, secret string) error {
	m[service+"/"+user] = secret
	return nil
}

func testSecretEnv(t *testing.T) func() (secretEnv, error) {
	dir := t.TempDir()
	env := secretEnv{
		paths:   vault.Paths{Store: filepath.Join(dir, "secrets.enc"), KeyFile: filepath.Join(dir, "secret.key")},
		keyring: memoryKeyring{},
	}
	return func() (secretEnv, error) { return env, nil }
}

func runSecret(t *testing.T, load func() (secretEnv, error), stdin string, args ...string) (string, error) {
	cmd := newSecretCmdWith(load)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSecretSetThenGet(t *testing.T) {
	load := testSecretEnv(t)
	if _, err := runSecret(t, load, "", "set", "GITHUB_TOKEN", "ghp_123"); err != nil {
		t.Fatal(err)
	}
	out, err := runSecret(t, load, "", "get", "GITHUB_TOKEN")
	if err != nil || out != "ghp_123\n" {
		t.Fatalf("get = %q, %v", out, err)
	}
}

func TestSecretSetReadsTheValueFromStdin(t *testing.T) {
	load := testSecretEnv(t)
	if _, err := runSecret(t, load, "piped-value\n", "set", "PIPED"); err != nil {
		t.Fatal(err)
	}
	if out, _ := runSecret(t, load, "", "get", "PIPED"); out != "piped-value\n" {
		t.Fatalf("get = %q", out)
	}
}

func TestSecretSetWithoutValueFails(t *testing.T) {
	if _, err := runSecret(t, testSecretEnv(t), "", "set", "EMPTY"); !errors.Is(err, errNoValue) {
		t.Fatalf("set = %v; want errNoValue", err)
	}
}

func TestSecretGetOfAMissingKeyFails(t *testing.T) {
	if _, err := runSecret(t, testSecretEnv(t), "", "get", "NOPE"); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("get = %v; want ErrNotFound", err)
	}
}

func TestSecretListMasksUnlessRevealed(t *testing.T) {
	load := testSecretEnv(t)
	runSecret(t, load, "", "set", "B", "abcd1234wxyz")
	runSecret(t, load, "", "set", "A", "short")
	masked, err := runSecret(t, load, "", "list")
	if err != nil || masked != "A\t*****\nB\tabcd****wxyz\n" {
		t.Fatalf("list = %q, %v", masked, err)
	}
	revealed, _ := runSecret(t, load, "", "list", "--reveal")
	if revealed != "A\tshort\nB\tabcd1234wxyz\n" {
		t.Fatalf("list --reveal = %q", revealed)
	}
}

func TestSecretConnectWithoutTokenFailsOutsideATerminal(t *testing.T) {
	t.Setenv(tokenEnv, "")
	if _, err := runSecret(t, testSecretEnv(t), "", "connect"); !errors.Is(err, errNoToken) {
		t.Fatalf("connect = %v; want errNoToken", err)
	}
}

func TestChooseProject(t *testing.T) {
	projects := []bitwarden.Project{{ID: "1", Name: "work"}, {ID: "2", Name: "personal"}}
	if p, err := chooseProject(projects, "personal", ""); err != nil || p.ID != "2" {
		t.Errorf("by name = %+v, %v", p, err)
	}
	if _, err := chooseProject(projects, "nope", "1"); err == nil {
		t.Error("an unknown --project fell back to the remembered one")
	}
	if p, err := chooseProject(projects, "", "1"); err != nil || p.ID != "1" {
		t.Errorf("remembered = %+v, %v", p, err)
	}
	if p, err := chooseProject(projects[:1], "", "gone"); err != nil || p.ID != "1" {
		t.Errorf("single project = %+v, %v", p, err)
	}
	if _, err := chooseProject(projects, "", ""); err == nil {
		t.Error("picked a project among many without a terminal")
	}
	if _, err := chooseProject(nil, "", ""); !errors.Is(err, errNoProjects) {
		t.Errorf("no projects = %v", err)
	}
}

func TestRemoteEntriesKeepsTheFirstOfDuplicateKeys(t *testing.T) {
	entries, duplicates := remoteEntries([]bitwarden.Secret{
		{ID: "1", Key: "A", Value: "first"},
		{ID: "2", Key: "A", Value: "second"},
		{ID: "3", Key: "B", Value: "b"},
	})
	if entries["A"] != (vault.Entry{Value: "first", ID: "1"}) || !reflect.DeepEqual(duplicates, []string{"A"}) {
		t.Fatalf("entries = %+v, duplicates = %v", entries, duplicates)
	}
}

func TestForgetRemoteIDs(t *testing.T) {
	got := forgetRemoteIDs(map[string]vault.Entry{"A": {Value: "a", ID: "old-project"}})
	if got["A"] != (vault.Entry{Value: "a"}) {
		t.Fatalf("got = %+v", got)
	}
}
