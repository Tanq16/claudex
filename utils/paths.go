package utils

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		PrintFatal("cannot resolve home directory", err)
	}
	return home
}

func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(HomeDir(), path[1:])
	}
	return path
}

func ResolveConfigDir(flag string) string {
	if flag != "" {
		return ExpandPath(flag)
	}
	return filepath.Join(HomeDir(), ".claude")
}

func DiscoverAccountPaths() []string {
	home := HomeDir()
	var paths []string

	entries, err := os.ReadDir(home)
	if err != nil {
		return []string{filepath.Join(home, ".claude")}
	}

	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() {
			continue
		}
		if name == ".claude" || numberedAccount(name) {
			paths = append(paths, filepath.Join(home, name))
		}
	}

	if len(paths) == 0 {
		return []string{filepath.Join(home, ".claude")}
	}

	slices.Sort(paths)
	return paths
}

func numberedAccount(name string) bool {
	suffix, ok := strings.CutPrefix(name, ".claude")
	if !ok || suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func ResolveAccountPaths(account string) []string {
	if account != "" {
		return []string{ExpandPath(account)}
	}
	return DiscoverAccountPaths()
}

func AbbreviatePath(path string) string {
	home := HomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func ClaudexConfigDir() string {
	return filepath.Join(HomeDir(), ".config", "claudex")
}

func GlobalPluginDir() string {
	return filepath.Join(ClaudexConfigDir(), "global")
}

func PresetsDir() string {
	return filepath.Join(ClaudexConfigDir(), "presets")
}

func RemotePresetsDir() string {
	return filepath.Join(ClaudexConfigDir(), "remote-presets")
}
