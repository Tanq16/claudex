package session

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tanq16/claudex/internal/accounts"
)

var ErrRunning = errors.New("a session is already running")

type Launch struct {
	Name       string
	Dir        string
	RCName     string
	Account    string
	Connectors bool
	Continue   bool
}

type Launcher struct {
	DefaultAccount string
	PluginDir      string
}

func connectorsSettings() string {
	data, _ := json.Marshal(map[string]any{
		"env": map[string]string{"ENABLE_CLAUDEAI_MCP_SERVERS": "true"},
	})
	return string(data)
}

func (l Launcher) Start(x Launch) error {
	argv, err := l.prepare(x)
	if err != nil {
		return err
	}
	windows, err := Windows()
	if err != nil {
		return err
	}
	if w, ok := windows[x.Name]; ok {
		if !w.Dead {
			return ErrRunning
		}
		if err := kill(x.Name); err != nil {
			return err
		}
	}
	return newWindow(x.Name, x.Dir, argv)
}

func (l Launcher) Restart(x Launch) error {
	argv, err := l.prepare(x)
	if err != nil {
		return err
	}
	windows, err := Windows()
	if err != nil {
		return err
	}
	if _, ok := windows[x.Name]; !ok {
		return newWindow(x.Name, x.Dir, argv)
	}
	return respawn(x.Name, x.Dir, argv)
}

func (l Launcher) prepare(x Launch) ([]string, error) {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude not found in PATH: %w", err)
	}
	if err := accounts.Trust(x.Account, l.DefaultAccount, x.Dir); err != nil {
		return nil, fmt.Errorf("trusting %s for the account: %w", x.Dir, err)
	}

	argv := []string{"env", "-u", "CLAUDE_CONFIG_DIR"}
	for _, kv := range accounts.Env(x.Account, l.DefaultAccount) {
		if strings.HasPrefix(kv, "CLAUDE_CONFIG_DIR=") || strings.HasPrefix(kv, "PATH=") {
			argv = append(argv, kv)
		}
	}
	argv = append(argv, claude)
	return append(argv, Args(x, l.PluginDir)...), nil
}

func Args(x Launch, pluginDir string) []string {
	var args []string
	if x.Continue {
		args = append(args, "--continue")
	}
	if x.RCName != "" {
		args = append(args, "--remote-control", x.RCName)
	}
	if x.Connectors {
		args = append(args, "--settings", connectorsSettings())
	}
	if pluginDir != "" {
		args = append(args, "--plugin-dir", pluginDir)
	}
	return append(args, "--dangerously-skip-permissions")
}
