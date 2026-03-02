package parser

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tanq16/claudex/internal/model"
)

type claudeJSON struct {
	OAuthAccount struct {
		EmailAddress     string `json:"emailAddress"`
		OrganizationName string `json:"organizationName"`
		DisplayName      string `json:"displayName"`
	} `json:"oauthAccount"`
}

func ParseAccount(configDir string) (model.AccountInfo, error) {
	jsonPath := filepath.Join(configDir, ".claude.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
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
