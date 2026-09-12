package tracker

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func ComputeAccountUsage(configDir, defaultDir string) (model.AccountUsage, error) {
	acct, err := parser.ParseAccount(configDir)
	if err != nil {
		acct = model.AccountInfo{ConfigDir: configDir, Email: "unknown"}
	}

	usage := model.AccountUsage{Account: acct}

	token, err := getOAuthToken(configDir, defaultDir)
	if err != nil {
		usage.Unavailable = "cannot read the stored credential"
		usage.Err = err
		return usage, nil
	}
	if token == "" {
		usage.TokenExpired = true
		return usage, nil
	}

	resp, err := fetchUsage(token)
	if err != nil {
		usage.Unavailable = "cannot reach the usage API"
		usage.Err = err
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

type claudeCredentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
	} `json:"claudeAiOauth"`
}

func getOAuthToken(configDir, defaultDir string) (string, error) {
	// The macOS Keychain value and the Linux/Windows .credentials.json file hold the same blob.
	var raw []byte
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("security", "find-generic-password", "-s", keychainServiceName(configDir, defaultDir), "-w")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			if detail := strings.TrimSpace(stderr.String()); detail != "" {
				return "", fmt.Errorf("reading the keychain: %s: %w", detail, err)
			}
			return "", fmt.Errorf("reading the keychain: %w", err)
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

func keychainServiceName(configDir, defaultDir string) string {
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
	if err := json.UnmarshalRead(resp.Body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
