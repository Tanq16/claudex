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
	Limits         []apiLimit `json:"limits"`
	Error          *apiError  `json:"error"`
}

type apiWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type apiLimit struct {
	Kind     string    `json:"kind"`
	Percent  float64   `json:"percent"`
	ResetsAt string    `json:"resets_at"`
	Scope    *apiScope `json:"scope"`
}

type apiScope struct {
	Model *struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
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

	usage.Windows = buildWindows(resp)
	return usage, nil
}

// The "limits" array is the authoritative per-model weekly source; fall back to the legacy fixed windows when it's absent.
func buildWindows(resp *apiResponse) []model.UsageWindow {
	if len(resp.Limits) > 0 {
		windows := make([]model.UsageWindow, 0, len(resp.Limits))
		for _, l := range resp.Limits {
			w := model.UsageWindow{Kind: l.Kind, Utilization: l.Percent, ResetsAt: parseTime(l.ResetsAt)}
			if l.Scope != nil && l.Scope.Model != nil {
				w.Scope = l.Scope.Model.DisplayName
			}
			windows = append(windows, w)
		}
		return windows
	}

	var windows []model.UsageWindow
	if resp.FiveHour != nil {
		windows = append(windows, legacyWindow("session", "", resp.FiveHour))
	}
	if resp.SevenDay != nil {
		windows = append(windows, legacyWindow("weekly_all", "", resp.SevenDay))
	}
	if resp.SevenDaySonnet != nil {
		windows = append(windows, legacyWindow("weekly_scoped", "Sonnet", resp.SevenDaySonnet))
	}
	return windows
}

func legacyWindow(kind, scope string, aw *apiWindow) model.UsageWindow {
	return model.UsageWindow{Kind: kind, Scope: scope, Utilization: aw.Utilization, ResetsAt: parseTime(aw.ResetsAt)}
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// claudeCredentials matches the OAuth blob Claude Code stores — the macOS Keychain value and the Linux/Windows .credentials.json file share this shape.
type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

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
