package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	u "github.com/tanq16/claudex/utils"
)

var initPluginCmd = &cobra.Command{
	Use:   "init-plugin <name>",
	Short: "Scaffold a Claude Code plugin directory",
	Args:  cobra.ExactArgs(1),
	Run:   runInitPlugin,
}

func runInitPlugin(cmd *cobra.Command, args []string) {
	name := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		u.PrintFatal("failed to resolve current directory", err)
	}
	target := filepath.Join(cwd, name)

	if _, err := os.Stat(target); err == nil {
		u.PrintFatal(fmt.Sprintf("%s already exists; refusing to overwrite", target), nil)
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		u.PrintFatal("failed to create plugin scaffold", err)
	}
	for _, sub := range []string{"skills", "agents", "hooks", "output-styles"} {
		dir := filepath.Join(target, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			u.PrintFatal("failed to create plugin scaffold", err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644); err != nil {
			u.PrintFatal("failed to create plugin scaffold", err)
		}
	}

	if err := os.MkdirAll(filepath.Join(target, ".claude-plugin"), 0o755); err != nil {
		u.PrintFatal("failed to create plugin scaffold", err)
	}

	files := []struct {
		path string
		body any
	}{
		{filepath.Join(target, ".claude-plugin", "plugin.json"), map[string]any{"name": name, "version": "0.0.1"}},
		{filepath.Join(target, ".mcp.json"), map[string]any{"mcpServers": map[string]any{}}},
		{filepath.Join(target, ".lsp.json"), map[string]any{}},
		{filepath.Join(target, "settings.json"), map[string]any{}},
	}
	for _, f := range files {
		if err := writeJSON(f.path, f.body); err != nil {
			u.PrintFatal("failed to create plugin scaffold", err)
		}
	}

	u.PrintSuccess(fmt.Sprintf("Created plugin scaffold at %s", target))
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
