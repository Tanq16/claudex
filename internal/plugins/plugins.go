package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Everything here loads into every session, which is why it carries language servers and nothing else; skills reach a project through `claudex apply`.
func BuildGlobalPlugin(dir string) error {
	if err := writeGlobalManifest(dir); err != nil {
		return err
	}
	return writeGlobalLSP(dir)
}

// Always rewritten (not write-if-missing) so a manifest from an older plugin name migrates to "claudex".
func writeGlobalManifest(dir string) error {
	manifest := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{
		"name":        "claudex",
		"description": "claudex's language servers, auto-loaded across every account",
		"version":     "0.0.1",
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(manifest, data, 0o644)
}

// Rewritten every build so an added server or schema change reaches existing installs. A server whose binary is absent is skipped by Claude Code, so shipping all three by default is safe.
func writeGlobalLSP(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{
		"go": map[string]any{
			"command":             "gopls",
			"args":                []string{"serve"},
			"extensionToLanguage": map[string]string{".go": "go"},
		},
		"python": map[string]any{
			"command":             "pyright-langserver",
			"args":                []string{"--stdio"},
			"extensionToLanguage": map[string]string{".py": "python", ".pyi": "python"},
		},
		"typescript": map[string]any{
			"command": "typescript-language-server",
			"args":    []string{"--stdio"},
			"extensionToLanguage": map[string]string{
				".ts": "typescript", ".mts": "typescript", ".cts": "typescript",
				".tsx": "typescriptreact",
				".js":  "javascript", ".mjs": "javascript", ".cjs": "javascript",
				".jsx": "javascriptreact",
			},
		},
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(filepath.Join(dir, ".lsp.json"), data, 0o644)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
