package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/auth"
	u "github.com/tanq16/claudex/utils"
)

var oauthTokenFlags struct {
	port      int
	expiresIn int
	manual    bool
}

var oauthTokenCmd = &cobra.Command{
	Use:   "oauth-token",
	Short: "Obtain a Claude OAuth access token via PKCE flow",
	Long:  "Opens a browser for Claude authentication using OAuth 2.0 PKCE flow and prints the access token to stdout.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		cfg := auth.Config{
			Port:      oauthTokenFlags.port,
			ExpiresIn: oauthTokenFlags.expiresIn,
			Manual:    oauthTokenFlags.manual,
		}

		token, err := auth.Login(ctx, cfg, openBrowser)
		if errors.Is(err, u.ErrNoTerminal) {
			u.PrintFatal("oauth-token --manual needs an interactive terminal to paste the code into", nil)
		}
		if err != nil {
			u.PrintFatal("OAuth flow failed", err)
		}

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
		return fmt.Errorf("no browser launcher for %s", runtime.GOOS)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			err = fmt.Errorf("%s: %w", detail, err)
		}
		return fmt.Errorf("%s: %w", cmd.Args[0], err)
	}
	return nil
}

func init() {
	oauthTokenCmd.Flags().IntVarP(&oauthTokenFlags.port, "port", "p", 0, "Local port for the OAuth callback server (0 picks a free one)")
	oauthTokenCmd.Flags().IntVarP(&oauthTokenFlags.expiresIn, "expires-in", "e", auth.DefaultExpiresIn, "Requested token expiry in seconds (server may override)")
	oauthTokenCmd.Flags().BoolVar(&oauthTokenFlags.manual, "manual", false, "Print the authorize URL and paste the code back, instead of running a callback server")
	oauthTokenCmd.MarkFlagsMutuallyExclusive("port", "manual")
}
