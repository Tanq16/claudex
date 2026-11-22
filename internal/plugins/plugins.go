package plugins

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/fsutil"
)

const (
	dirMode  = 0o700
	fileMode = 0o600
)

func BuildGlobalPlugin(dir string) error {
	if err := writeGlobalManifest(dir); err != nil {
		return err
	}
	return writeGlobalLSP(dir)
}

func writeGlobalManifest(dir string) error {
	manifest := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(manifest), dirMode); err != nil {
		return err
	}
	data, err := json.Marshal(map[string]any{
		"name":        "claudex",
		"description": "claudex's language servers, auto-loaded across every account",
		"version":     "0.0.1",
	}, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(manifest, data, fileMode)
}

func writeGlobalLSP(dir string) error {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	// Claude Code skips a server whose binary is absent, so all three ship unconditionally.
	data, err := json.Marshal(map[string]any{
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
	}, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return fsutil.WriteFileAtomic(filepath.Join(dir, ".lsp.json"), data, fileMode)
}

func PruneGlobal(dir string, legacySkills fs.FS, skillsRoot string) {
	os.Remove(filepath.Join(dir, "output-styles", "claudex.md"))
	os.Remove(filepath.Join(dir, "output-styles"))

	entries, err := fs.ReadDir(legacySkills, skillsRoot)
	if err != nil {
		return
	}
	for _, e := range entries {
		os.RemoveAll(filepath.Join(dir, "skills", e.Name()))
	}
	os.Remove(filepath.Join(dir, "skills"))
}
