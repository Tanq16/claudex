package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/oauth"
	u "github.com/tanq16/claudex/utils"
)

var oauthTokenFlags struct {
	port      int
	expiresIn int
}

var oauthTokenCmd = &cobra.Command{
	Use:   "oauth-token",
	Short: "Obtain a Claude OAuth access token via PKCE flow",
	Long:  "Opens a browser for Claude authentication using OAuth 2.0 PKCE flow and prints the access token to stdout.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		cfg := oauth.Config{
			Port:      oauthTokenFlags.port,
			ExpiresIn: oauthTokenFlags.expiresIn,
		}

		u.PrintInfo(fmt.Sprintf("Starting OAuth flow on port %d (requested expiry: %ds)", cfg.Port, cfg.ExpiresIn))
		u.PrintInfo("Opening browser for Claude authentication...")

		token, err := oauth.RunFlow(ctx, cfg, openBrowser)
		if err != nil {
			u.PrintFatal("OAuth flow failed", err)
		}

		u.PrintSuccess("Authentication successful.")
		u.PrintGeneric(token)
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		u.PrintWarn("Cannot auto-open browser. Open this URL manually:", nil)
		u.PrintGeneric(url)
		return nil
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", cmd.Args[0], err)
	}
	return nil
}

func init() {
	oauthTokenCmd.Flags().IntVarP(&oauthTokenFlags.port, "port", "p", oauth.DefaultPort, "Local port for OAuth callback server")
	oauthTokenCmd.Flags().IntVarP(&oauthTokenFlags.expiresIn, "expires-in", "e", oauth.DefaultExpiresIn, "Requested token expiry in seconds (server may override)")
}
