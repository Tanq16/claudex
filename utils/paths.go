package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ExpandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}

func ResolveConfigDir(flag string) string {
	if flag != "" {
		return ExpandPath(flag)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func DiscoverAccountPaths() []string {
	home, _ := os.UserHomeDir()
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
		if name == ".claude" {
			paths = append(paths, filepath.Join(home, name))
			continue
		}
		if strings.HasPrefix(name, ".claude") {
			suffix := name[len(".claude"):]
			if _, err := strconv.Atoi(suffix); err == nil {
				paths = append(paths, filepath.Join(home, name))
			}
		}
	}

	if len(paths) == 0 {
		return []string{filepath.Join(home, ".claude")}
	}

	sort.Strings(paths)
	return paths
}

func ResolveAccountPaths(account string) []string {
	if account != "" {
		return []string{ExpandPath(account)}
	}
	return DiscoverAccountPaths()
}
