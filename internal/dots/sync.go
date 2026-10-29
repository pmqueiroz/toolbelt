package dots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/pmqueiroz/toolbelt/internal/files"
	"github.com/pmqueiroz/toolbelt/internal/merge"
	"github.com/pmqueiroz/toolbelt/internal/plan"
)

var ErrNotManaged = errors.New("dot files are not managed by tb yet, run tb setup")

type storedVars struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type state struct {
	upstream   []byte
	base       []byte
	hasBase    bool
	local      []byte
	hasLocal   bool
	linked     bool
	targetFile bool
}

type outcome struct {
	content   []byte
	backup    bool
	conflicts int
}

func LoadVars(paths Paths) (Vars, error) {
	data, err := os.ReadFile(paths.vars())
	if errors.Is(err, fs.ErrNotExist) {
		return Vars{}, ErrNotManaged
	}
	if err != nil {
		return Vars{}, err
	}
	var stored storedVars
	if err := json.Unmarshal(data, &stored); err != nil {
		return Vars{}, fmt.Errorf("read %s: %w", paths.vars(), err)
	}
	return Vars{Username: stored.Username, Email: stored.Email}, nil
}

func Plan(goos string, paths Paths, vars Vars, now time.Time) ([]plan.Action, error) {
	var actions []plan.Action
	if action, ok, err := saveVarsAction(paths, vars); err != nil {
		return nil, err
	} else if ok {
		actions = append(actions, action)
	}

	for _, d := range All {
		if len(d.OS) > 0 && !slices.Contains(d.OS, goos) {
			continue
		}
		action, ok, err := dotAction(d, paths, vars, now)
		if err != nil {
			return nil, err
		}
		if ok {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

func saveVarsAction(paths Paths, vars Vars) (plan.Action, bool, error) {
	content, err := json.MarshalIndent(storedVars{Username: vars.Username, Email: vars.Email}, "", "  ")
	if err != nil {
		return plan.Action{}, false, err
	}
	content = append(content, '\n')
	current, err := os.ReadFile(paths.vars())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return plan.Action{}, false, err
	}
	if bytes.Equal(current, content) {
		return plan.Action{}, false, nil
	}
	return plan.Action{
		Description: fmt.Sprintf("save git identity for dot file sync to %s", files.Pretty(paths.vars(), paths.Home)),
		Run: func(context.Context) error {
			return writeFile(paths.vars(), content)
		},
	}, true, nil
}

func dotAction(d Dot, paths Paths, vars Vars, now time.Time) (plan.Action, bool, error) {
	raw, err := embedded.ReadFile("files/" + d.File)
	if err != nil {
		return plan.Action{}, false, err
	}
	target, live, base := d.Target(paths), paths.live(d.File), paths.base(d.File)

	s, err := readState(target, live, base)
	if err != nil {
		return plan.Action{}, false, err
	}
	s.upstream = []byte(Render(string(raw), vars))

	if (s.linked || s.targetFile) && s.hasLocal && s.hasBase && bytes.Equal(s.base, s.upstream) {
		return plan.Action{}, false, nil
	}

	pretty := func(p string) string { return files.Pretty(p, paths.Home) }
	if s.hasBase && merge.HasMarkers(s.local) {
		return plan.Action{
			Description: fmt.Sprintf("skip %s: unresolved conflict markers", pretty(live)),
			Run: func(context.Context) error {
				return fmt.Errorf("resolve the conflict markers in %s, then run tb dots sync", pretty(live))
			},
		}, true, nil
	}

	o := decide(s)
	backup := files.BackupPath(target, now)
	description := describe(d.File, pretty(target), pretty(live), s, o)
	if o.backup {
		description += fmt.Sprintf(" (backup %s)", pretty(backup))
	}

	return plan.Action{
		Description: description,
		Run: func(context.Context) error {
			if o.backup {
				if err := writeFile(backup, s.local); err != nil {
					return err
				}
			}
			if err := writeFile(live, o.content); err != nil {
				return err
			}
			if err := writeFile(base, s.upstream); err != nil {
				return err
			}
			if !s.linked {
				if err := link(target, live, o.content); err != nil {
					return err
				}
			}
			if o.conflicts > 0 {
				return fmt.Errorf("%d conflict(s) in %s, resolve the <<<<<<< markers (your version is in %s)", o.conflicts, pretty(live), pretty(backup))
			}
			return nil
		},
	}, true, nil
}

func decide(s state) outcome {
	switch {
	case !s.hasLocal, bytes.Equal(s.local, s.upstream):
		return outcome{content: s.upstream}
	case !s.hasBase:
		return outcome{content: s.upstream, backup: true}
	}
	r := merge.Three(s.base, s.local, s.upstream)
	return outcome{
		content:   r.Content,
		conflicts: r.Conflicts,
		backup:    r.Conflicts > 0 || (s.targetFile && !bytes.Equal(r.Content, s.local)),
	}
}

func describe(file, target, live string, s state, o outcome) string {
	switch {
	case o.conflicts > 0:
		return fmt.Sprintf("merge %s into %s with %d conflict(s) to resolve", file, live, o.conflicts)
	case !s.hasLocal:
		return fmt.Sprintf("write %s to %s and link %s to it", file, live, target)
	case !s.linked && s.hasBase && !bytes.Equal(s.local, s.base):
		return fmt.Sprintf("merge %s with local edits from %s and link %s to %s", file, target, target, live)
	case !s.linked:
		return fmt.Sprintf("replace %s with a link to %s", target, live)
	case bytes.Equal(s.local, s.base) || bytes.Equal(s.local, s.upstream):
		return fmt.Sprintf("update %s", live)
	default:
		return fmt.Sprintf("merge %s into %s, keeping local edits", file, live)
	}
}

func readState(target, live, base string) (state, error) {
	var s state
	var err error
	s.base, s.hasBase, err = readOptional(base)
	if err != nil {
		return s, err
	}

	s.linked = linksTo(target, live)
	info, err := os.Stat(target)
	switch {
	case s.linked || errors.Is(err, fs.ErrNotExist):
		s.local, s.hasLocal, err = readOptional(live)
		return s, err
	case err != nil:
		return s, err
	case info.IsDir():
		return s, fmt.Errorf("%s is a directory", target)
	}
	s.targetFile = true
	s.local, err = os.ReadFile(target)
	s.hasLocal = err == nil
	return s, err
}

func readOptional(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return content, err == nil, err
}

func linksTo(target, live string) bool {
	dest, err := os.Readlink(target)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(filepath.Dir(target), dest)
	}
	return filepath.Clean(dest) == filepath.Clean(live)
}

func link(target, live string, content []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".%s.tblink", filepath.Base(target)))
	os.Remove(tmp)
	if err := os.Symlink(live, tmp); err != nil {
		return files.WriteAtomic(target, content, 0o644)
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return files.WriteAtomic(path, content, 0o644)
}
