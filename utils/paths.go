package utils

import (
	"os"

	"github.com/tanq16/claudex/internal/paths"
)

func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		PrintFatal("cannot resolve home directory", err)
	}
	return home
}

func ExpandPath(path string) string {
	return paths.Expand(HomeDir(), path)
}

func ResolveConfigDir(flag string) string {
	if flag != "" {
		return ExpandPath(flag)
	}
	return paths.DefaultAccount(HomeDir())
}

func DiscoverAccountPaths() []string {
	return paths.DiscoverAccounts(HomeDir())
}

func AbbreviatePath(path string) string {
	return paths.Abbreviate(HomeDir(), path)
}

func ClaudexConfigDir() string {
	return paths.ConfigDir(HomeDir())
}

func GlobalPluginDir() string {
	return paths.GlobalPluginDir(HomeDir())
}

func PresetsDir() string {
	return paths.PresetsDir(HomeDir())
}

func BaseSkillsDir() string {
	return paths.BaseSkillsDir(HomeDir())
}

func RemotePresetsDir() string {
	return paths.RemotePresetsDir(HomeDir())
}

func ResumeStash(projectDir string) string {
	return paths.ResumeStash(HomeDir(), projectDir)
}

func DefaultsFile() string {
	return paths.DefaultsFile(HomeDir())
}
