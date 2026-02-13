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

	taskCmd "github.com/tanq16/claude-usage/cmd/task-cmd"
)

var AppVersion = "dev-build"

var debugFlag bool
var extraAccounts []string

// AccountPaths is the resolved list of account config directories
var AccountPaths []string

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

func buildAccountPaths() {
	home, _ := os.UserHomeDir()
	AccountPaths = []string{filepath.Join(home, ".claude")}
	for _, p := range extraAccounts {
		if len(p) > 0 && p[0] == '~' {
			p = filepath.Join(home, p[1:])
		}
		AccountPaths = append(AccountPaths, p)
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringSliceVar(&extraAccounts, "accounts", []string{}, "Additional Claude config directories to monitor")

	cobra.OnInitialize(setupLogs, buildAccountPaths)

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(taskCmd.TaskCmd)
}
