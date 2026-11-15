package cmd

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	u "github.com/tanq16/claudex/utils"
)

var switchFlags struct {
	account string
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move every session of the current directory into one account",
	Args:  cobra.NoArgs,
	Run:   runSwitch,
}

func init() {
	switchCmd.Flags().StringVarP(&switchFlags.account, "account", "A", "", "Account to move this directory's sessions into (required)")
	switchCmd.MarkFlagRequired("account")
}

func runSwitch(cmd *cobra.Command, args []string) {
	cwd := currentDir()
	target := resolveAccount(switchFlags.account)

	moved := 0
	for _, source := range u.DiscoverAccountPaths() {
		if source == target {
			continue
		}
		n, err := convo.MoveProject(cwd, source, target)
		if err != nil {
			u.PrintFatal("Failed to move sessions out of "+u.AbbreviatePath(source), err)
		}
		if n > 0 {
			u.PrintIndentedSuccess(fmt.Sprintf("%d session(s) from %s", n, u.AbbreviatePath(source)))
		}
		moved += n
	}
	if moved == 0 {
		u.PrintSuccess("No sessions outside " + u.AbbreviatePath(target) + "; nothing to switch")
		return
	}
	u.PrintSuccess(fmt.Sprintf("Moved %d session(s) into %s", moved, u.AbbreviatePath(target)))
}

func resolveAccount(flag string) string {
	discovered := u.DiscoverAccountPaths()
	expanded := filepath.Clean(u.ExpandPath(flag))
	if i := slices.IndexFunc(discovered, func(a string) bool {
		return filepath.Clean(a) == expanded || filepath.Base(a) == flag
	}); i >= 0 {
		return discovered[i]
	}
	u.PrintFatal(fmt.Sprintf("account %q matches no discovered account", flag), nil)
	return ""
}
