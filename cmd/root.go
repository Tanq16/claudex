package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/claude-usage/internal/utils"

	pluginCmd "github.com/tanq16/claude-usage/cmd/plugin-cmd"
	taskCmd "github.com/tanq16/claude-usage/cmd/task-cmd"
)

// ResolveAccountPaths returns the default ~/.claude path plus any extra
// account directories supplied via the per-command --accounts flag.
func ResolveAccountPaths(extra []string) []string {
	home, _ := os.UserHomeDir()
	paths := []string{filepath.Join(home, ".claude")}
	for _, p := range extra {
		if len(p) > 0 && p[0] == '~' {
			p = filepath.Join(home, p[1:])
		}
		paths = append(paths, p)
	}
	return paths
}

var AppVersion = "dev-build"

var debugFlag bool

var rootCmd = &cobra.Command{
	Use:     "claude-usage",
	Short:   "Monitor and plan Claude Code usage across accounts",
	Version: AppVersion,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupLogs() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
		NoColor:    false,
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debugFlag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		utils.GlobalDebugFlag = true
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")

	cobra.OnInitialize(setupLogs)

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(convosCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(taskCmd.TaskCmd)
	rootCmd.AddCommand(pluginCmd.PluginCmd)
}
