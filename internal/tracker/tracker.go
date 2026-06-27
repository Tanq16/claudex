package tracker

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/tanq16/claudex/internal/model"
	"github.com/tanq16/claudex/internal/parser"
)

type apiResponse struct {
	FiveHour       *apiWindow `json:"five_hour"`
	SevenDay       *apiWindow `json:"seven_day"`
	SevenDaySonnet *apiWindow `json:"seven_day_sonnet"`
	Error          *apiError  `json:"error"`
}

type apiWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func ComputeAccountUsage(configDir string) (model.AccountUsage, error) {
	acct, err := parser.ParseAccount(configDir)
	if err != nil {
		acct = model.AccountInfo{ConfigDir: configDir, Email: "unknown"}
	}

	usage := model.AccountUsage{Account: acct}

	token, err := getOAuthToken(configDir)
	if err != nil || token == "" {
		usage.TokenExpired = true
		return usage, nil
	}

	resp, err := fetchUsage(token)
	if err != nil {
		usage.TokenExpired = true
		return usage, nil
	}

	if resp.Error != nil {
		if resp.Error.Type == "authentication_error" || resp.Error.Type == "permission_error" {
			usage.TokenExpired = true
		} else {
			usage.APIError = resp.Error.Message
		}
		return usage, nil
	}

	if resp.FiveHour != nil {
		w := parseWindow(resp.FiveHour)
		usage.FiveHour = &w
	}
	if resp.SevenDay != nil {
		w := parseWindow(resp.SevenDay)
		usage.SevenDay = &w
	}
	if resp.SevenDaySonnet != nil {
		w := parseWindow(resp.SevenDaySonnet)
		usage.SevenDaySonnet = &w
	}

	return usage, nil
}

func parseWindow(aw *apiWindow) model.UsageWindow {
	w := model.UsageWindow{Utilization: aw.Utilization}
	if aw.ResetsAt != "" {
		t, err := time.Parse(time.RFC3339Nano, aw.ResetsAt)
		if err == nil {
			w.ResetsAt = t
		}
	}
	return w
}

// claudeCredentials matches the OAuth blob Claude Code stores — the macOS Keychain value and the Linux/Windows .credentials.json file share this shape.
type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

// getOAuthToken reads the account's OAuth token: macOS from the Keychain, Linux/Windows from a .credentials.json file inside the config dir.
func getOAuthToken(configDir string) (string, error) {
	var raw []byte
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("security", "find-generic-password", "-s", keychainServiceName(configDir), "-w").Output()
		if err != nil {
			return "", err
		}
		raw = bytes.TrimSpace(out)
	} else {
		data, err := os.ReadFile(filepath.Join(configDir, ".credentials.json"))
		if err != nil {
			return "", err
		}
		raw = data
	}

	var creds claudeCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return "", err
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("no access token found")
	}
	return creds.ClaudeAiOauth.AccessToken, nil
}

func keychainServiceName(configDir string) string {
	home, _ := os.UserHomeDir()
	defaultDir := filepath.Join(home, ".claude")

	if configDir == defaultDir {
		return "Claude Code-credentials"
	}

	h := sha256.Sum256([]byte(configDir))
	suffix := fmt.Sprintf("%x", h[:4])
	return "Claude Code-credentials-" + suffix
}

func fetchUsage(token string) (*apiResponse, error) {
	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "claude-code/2.1.34")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
