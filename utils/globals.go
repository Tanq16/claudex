package utils

import (
	"os"
	"path/filepath"
)

var GlobalDebugFlag bool
var GlobalForAIFlag bool

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

func ResolveAccountPaths(extra []string) []string {
	home, _ := os.UserHomeDir()
	paths := []string{filepath.Join(home, ".claude")}
	for _, p := range extra {
		paths = append(paths, ExpandPath(p))
	}
	return paths
}
