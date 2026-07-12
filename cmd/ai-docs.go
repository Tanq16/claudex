package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	u "github.com/tanq16/claudex/utils"
)

var aiDocsFlags struct {
	port int
	docs string
}

var aiDocsCmd = &cobra.Command{
	Use:   "ai-docs",
	Short: "Serve the ai-docs viewer for the current project",
	Run:   runAIDocs,
}

func runAIDocs(cmd *cobra.Command, args []string) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		u.PrintFatal("node not found in PATH; ai-docs needs Node.js installed", err)
	}

	serverPath := filepath.Join(u.GlobalPluginDir(), "skills", "ai-docs", "assets", "server.mjs")
	if _, err := os.Stat(serverPath); err != nil {
		u.PrintFatal("ai-docs viewer not found; run `claudex configure` first", err)
	}

	// Only forward flags the user set, so server.mjs keeps ownership of the defaults.
	argv := []string{"node", serverPath}
	if aiDocsFlags.docs != "" {
		argv = append(argv, "--docs", aiDocsFlags.docs)
	}
	if aiDocsFlags.port != 0 {
		argv = append(argv, "--port", strconv.Itoa(aiDocsFlags.port))
	}

	if err := syscall.Exec(nodePath, argv, os.Environ()); err != nil {
		u.PrintFatal("Failed to exec node", err)
	}
}

func init() {
	aiDocsCmd.Flags().StringVarP(&aiDocsFlags.docs, "docs", "d", "", "Directory to serve (default ./AI-docs)")
	aiDocsCmd.Flags().IntVarP(&aiDocsFlags.port, "port", "p", 0, "Port for the viewer (default 4321)")
}
