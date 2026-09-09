package cmd

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/utils"
)

var AppVersion = "dev-build"

var debugFlag bool

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
		os.Exit(1)
	}
}

func setupLogs() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	var out io.Writer = os.Stdout
	if utils.StdoutIsTerminal {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	}
	log.Logger = zerolog.New(out).With().Timestamp().Logger()
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
	rootCmd.AddCommand(oauthTokenCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(launchCmd)
	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(applyPresetCmd)
	rootCmd.AddCommand(createPresetCmd)
	rootCmd.AddCommand(pullPresetCmd)
	rootCmd.AddCommand(cleanCwdCmd)
}
