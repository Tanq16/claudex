package cmd

import (
	"maps"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/charmbracelet/x/term"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/server"
)

var serveFlags struct {
	host string
	port int
	root string
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the web UI that manages the projects under a root directory and their Claude sessions",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		srv := server.New(server.Config{Host: serveFlags.host, Port: serveFlags.port, Root: serveFlags.root})
		if err := srv.Setup(); err != nil {
			log.Fatal().Err(err).Msg("failed to set up server")
		}
		errc := make(chan error, 1)
		go func() { errc <- srv.Run() }()

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		interactive := term.IsTerminal(os.Stdin.Fd())
		warned := false
		for {
			select {
			case err := <-errc:
				srv.Shutdown()
				log.Fatal().Err(err).Msg("server error")
			case sig := <-signals:
				busy := srv.Busy()
				if sig == syscall.SIGTERM || !interactive || len(busy) == 0 || warned {
					srv.Shutdown()
					log.Info().Msg("stopped")
					return
				}
				for _, name := range slices.Sorted(maps.Keys(busy)) {
					log.Warn().Str("session", name).Strs("busy", busy[name]).Msg("attached session is busy")
				}
				log.Warn().Msg("press Ctrl+C again to stop these sessions and exit")
				warned = true
			}
		}
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveFlags.host, "host", "H", "127.0.0.1", "Host to bind to")
	serveCmd.Flags().IntVarP(&serveFlags.port, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringVarP(&serveFlags.root, "root", "r", "", "Directory whose subdirectories are the projects (default ~/.config/claudex/projects)")
}
