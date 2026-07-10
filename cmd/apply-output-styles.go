package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/embedded"
	u "github.com/tanq16/claudex/utils"
)

var applyOutputStylesCmd = &cobra.Command{
	Use:   "apply-output-styles",
	Short: "Install the embedded output style(s) into the current project",
	Run:   runApplyOutputStyles,
}

func runApplyOutputStyles(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}

	dest := filepath.Join(cwd, ".claude", "output-styles")
	count, err := writeOutputStyles(dest)
	if err != nil {
		u.PrintFatal("failed to install output styles", err)
	}
	u.PrintSuccess(fmt.Sprintf("Installed %d output style(s) into %s (enable with /config)", count, u.AbbreviatePath(dest)))
}

func writeOutputStyles(dest string) (int, error) {
	entries, err := fs.ReadDir(embedded.OutputStylesFS, "output-styles")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := embedded.OutputStylesFS.ReadFile("output-styles/" + entry.Name())
		if err != nil {
			return count, err
		}
		if err := os.WriteFile(filepath.Join(dest, entry.Name()), data, 0o644); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
