package pkgmgr

import (
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

var environmentKeys = []struct {
	root registry.Key
	path string
}{
	{registry.CURRENT_USER, `Environment`},
	{registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`},
}

func persistedPath() []string {
	var dirs []string
	for _, k := range environmentKeys {
		key, err := registry.OpenKey(k.root, k.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		value, kind, err := key.GetStringValue("Path")
		key.Close()
		if err != nil {
			continue
		}
		if kind == registry.EXPAND_SZ {
			if expanded, err := registry.ExpandString(value); err == nil {
				value = expanded
			}
		}
		dirs = append(dirs, filepath.SplitList(value)...)
	}
	return dirs
}
