package parser

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tanq16/claude-usage/internal/model"
)

type claudeJSON struct {
	OAuthAccount struct {
		EmailAddress     string `json:"emailAddress"`
		OrganizationName string `json:"organizationName"`
		DisplayName      string `json:"displayName"`
	} `json:"oauthAccount"`
}

func ParseAccount(configDir string) (model.AccountInfo, error) {
	// Try inside config dir first (e.g. ~/.claude2/.claude.json)
	jsonPath := filepath.Join(configDir, ".claude.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		// Fall back to sibling path (e.g. ~/.claude -> ~/.claude.json)
		jsonPath = configDir + ".json"
		data, err = os.ReadFile(jsonPath)
		if err != nil {
			return model.AccountInfo{ConfigDir: configDir}, err
		}
	}

	var cj claudeJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return model.AccountInfo{ConfigDir: configDir}, err
	}

	return model.AccountInfo{
		Email:        cj.OAuthAccount.EmailAddress,
		Organization: cj.OAuthAccount.OrganizationName,
		DisplayName:  cj.OAuthAccount.DisplayName,
		ConfigDir:    configDir,
	}, nil
}
