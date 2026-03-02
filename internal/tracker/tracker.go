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
	"time"

	"github.com/tanq16/claudex/internal/model"
	"github.com/tanq16/claudex/internal/parser"
)

type apiResponse struct {
	FiveHour       *apiWindow  `json:"five_hour"`
	SevenDay       *apiWindow  `json:"seven_day"`
	SevenDaySonnet *apiWindow  `json:"seven_day_sonnet"`
	Error          *apiError   `json:"error"`
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
		usage.TokenExpired = true
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

func getOAuthToken(configDir string) (string, error) {
	serviceName := keychainServiceName(configDir)

	out, err := exec.Command("security", "find-generic-password", "-s", serviceName, "-w").Output()
	if err != nil {
		return "", err
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &creds); err != nil {
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

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
