package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/utils"
)

var AppVersion = "dev-build"

var debugFlag bool
var forAIFlag bool

var rootCmd = &cobra.Command{
	Use:     "claudex",
	Short:   "Monitor Claude Code usage across accounts",
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
	if forAIFlag {
		zerolog.SetGlobalLevel(zerolog.Disabled)
		utils.GlobalForAIFlag = true
	}
}

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&forAIFlag, "for-ai", false, "AI-friendly output (plain text, no colors)")
	rootCmd.MarkFlagsMutuallyExclusive("debug", "for-ai")

	cobra.OnInitialize(setupLogs)

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(oauthTokenCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(statuslineCmd)
	rootCmd.AddCommand(applySkillsCmd)
}
