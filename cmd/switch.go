package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanq16/claudex/internal/convo"
	u "github.com/tanq16/claudex/utils"
)

var switchFlags struct {
	id   string
	from string
	to   string
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Move a conversation from one account to another",
	Run:   runSwitch,
}

func runSwitch(cmd *cobra.Command, args []string) {
	if switchFlags.id == "" {
		u.PrintFatal("--id is required", nil)
	}
	if switchFlags.to == "" {
		u.PrintFatal("--to is required", nil)
	}

	fromDir := u.ResolveConfigDir(switchFlags.from)
	toDir := u.ExpandPath(switchFlags.to)

	sf, err := convo.FindSession(fromDir, switchFlags.id)
	if err != nil {
		u.PrintFatal("Error searching source account", err)
	}
	if sf == nil {
		u.PrintFatal(fmt.Sprintf("Session %s not found in %s", switchFlags.id, fromDir), nil)
	}

	projectPath := sf.ProjectPath
	dstProjectDir := convo.ProjectDir(toDir, projectPath)

	if err := convo.MoveSession(switchFlags.id, sf.ProjectDir, dstProjectDir); err != nil {
		u.PrintFatal("Failed to move session files", err)
	}

	srcEntries, err := convo.ReadRawHistory(fromDir)
	if err != nil {
		u.PrintWarn("Could not read source history", err)
	} else {
		matching, rest := convo.FilterBySession(srcEntries, switchFlags.id)
		if len(matching) > 0 {
			if err := convo.AppendRawHistory(toDir, matching); err != nil {
				u.PrintWarn("Could not append to target history", err)
			}
			if err := convo.WriteRawHistory(fromDir, rest); err != nil {
				u.PrintWarn("Could not update source history", err)
			}
		}
	}

	u.PrintSuccess(fmt.Sprintf("Switched session %s from %s to %s", switchFlags.id, fromDir, toDir))
}

func init() {
	switchCmd.Flags().StringVar(&switchFlags.id, "id", "", "Session UUID to switch")
	switchCmd.Flags().StringVar(&switchFlags.from, "from", "", "Source config directory (default ~/.claude)")
	switchCmd.Flags().StringVar(&switchFlags.to, "to", "", "Target config directory")
	switchCmd.MarkFlagRequired("id")
	switchCmd.MarkFlagRequired("to")
}
