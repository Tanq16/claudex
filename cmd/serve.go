package cmd

import (
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
		if err := srv.Run(); err != nil {
			log.Fatal().Err(err).Msg("server error")
		}
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveFlags.host, "host", "H", "127.0.0.1", "Host to bind to")
	serveCmd.Flags().IntVarP(&serveFlags.port, "port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringVarP(&serveFlags.root, "root", "r", ".", "Directory whose subdirectories are the projects")
}
