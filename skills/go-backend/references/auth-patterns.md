# Authentication Patterns

OAuth2 authentication for CLI tools with three login modes and token persistence.

**Applies to: CLI Only projects.** Uses `utils/` for output and input. Resides in `internal/auth/`.

## Login Modes

| Mode | Flag | How It Works | Environment |
|------|------|-------------|-------------|
| Callback (default) | (none) | Opens browser, localhost server receives redirect | Interactive desktop |
| Device | `--device-login` | Shows URL + code, polls until authorized | Headless / SSH / server |
| Manual | `--manual` | Shows URL, user pastes authorization code | Last resort / no device flow support |

## internal/auth/auth.go

```go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
    "time"

    u "github.com/[GITHUB_USER]/[REPO_NAME]/utils"
    "golang.org/x/oauth2"
)

func ConfigDir() string {
    home, err := os.UserHomeDir()
    if err != nil {
        u.PrintFatal("cannot determine home directory", err)
    }
    dir := filepath.Join(home, ".config", "[APP_NAME]")
    if err := os.MkdirAll(dir, 0700); err != nil {
        u.PrintFatal("cannot create config directory", err)
    }
    return dir
}

func LoadCredentials() (*oauth2.Config, error) {
    credPath := filepath.Join(ConfigDir(), "credentials.json")
    data, err := os.ReadFile(credPath)
    if err != nil {
        return nil, fmt.Errorf("create %s with your OAuth client credentials", credPath)
    }
    // Parse credentials and configure scopes per provider
    // Google: google.ConfigFromJSON(data, scopes...)
    // Other providers: manual oauth2.Config construction
    return config, nil
}

func Login(config *oauth2.Config, mode string) (*oauth2.Token, error) {
    switch mode {
    case "device":
        return loginWithDevice(config)
    case "manual":
        state, err := generateState()
        if err != nil {
            return nil, fmt.Errorf("failed to generate state: %w", err)
        }
        return loginWithManual(config, state)
    default:
        state, err := generateState()
        if err != nil {
            return nil, fmt.Errorf("failed to generate state: %w", err)
        }
        return loginWithCallback(config, state)
    }
}

func loginWithCallback(config *oauth2.Config, state string) (*oauth2.Token, error) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        return nil, fmt.Errorf("cannot start callback server: %w", err)
    }
    port := listener.Addr().(*net.TCPAddr).Port

    config.RedirectURL = fmt.Sprintf("http://localhost:%d", port)

    authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

    codeCh := make(chan string, 1)
    errCh := make(chan error, 1)

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Query().Get("state") != state {
            errCh <- fmt.Errorf("state mismatch — possible CSRF attack")
            http.Error(w, "State mismatch", http.StatusBadRequest)
            return
        }
        code := r.URL.Query().Get("code")
        if code == "" {
            errCh <- fmt.Errorf("no auth code in callback")
            http.Error(w, "Missing code", http.StatusBadRequest)
            return
        }
        fmt.Fprint(w, "<html><body><h2>Authentication successful!</h2><p>You can close this tab.</p></body></html>")
        codeCh <- code
    })

    srv := &http.Server{Handler: mux}
    go func() {
        if err := srv.Serve(listener); err != http.ErrServerClosed {
            errCh <- err
        }
    }()

    u.PrintInfo("Opening browser for authentication...")
    if err := openBrowser(authURL); err != nil {
        srv.Close()
        return nil, fmt.Errorf("cannot open browser — use 'login --device-login' for headless environments")
    }
    u.PrintInfo("Waiting for authorization in browser...")
    u.PrintGeneric(authURL)

    var code string
    select {
    case code = <-codeCh:
    case err := <-errCh:
        srv.Close()
        return nil, err
    case <-time.After(5 * time.Minute):
        srv.Close()
        return nil, fmt.Errorf("authentication timed out")
    }

    srv.Close()

    token, err := config.Exchange(context.Background(), code)
    if err != nil {
        return nil, fmt.Errorf("token exchange failed: %w", err)
    }

    if err := SaveToken(token); err != nil {
        return nil, err
    }
    return token, nil
}

func loginWithDevice(config *oauth2.Config) (*oauth2.Token, error) {
    config.Endpoint.DeviceAuthURL = "[DEVICE_AUTH_URL]"

    da, err := config.DeviceAuth(context.Background())
    if err != nil {
        return nil, fmt.Errorf("device authorization failed: %w", err)
    }

    u.PrintInfo("To authenticate, visit the URL below and enter the code:")
    u.PrintGeneric(fmt.Sprintf("  URL:  %s", da.VerificationURI))
    u.PrintGeneric(fmt.Sprintf("  Code: %s", da.UserCode))
    u.PrintGeneric("")
    u.PrintInfo("Waiting for authorization...")

    token, err := config.DeviceAccessToken(context.Background(), da)
    if err != nil {
        return nil, fmt.Errorf("device token exchange failed: %w", err)
    }

    if err := SaveToken(token); err != nil {
        return nil, err
    }
    return token, nil
}

func loginWithManual(config *oauth2.Config, state string) (*oauth2.Token, error) {
    config.RedirectURL = "http://localhost"

    // NOTE: state is included because some providers require it, but it is NOT a CSRF
    // control here — the user copies the code out-of-band, so there is no automated redirect
    // whose state we can compare against. CSRF validation only happens in loginWithCallback.
    authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

    u.PrintInfo("Visit this URL to authenticate:")
    u.PrintGeneric(authURL)
    u.PrintGeneric("")
    u.PrintInfo("After authorizing, copy the 'code' parameter from the redirect URL.")

    code, err := u.PromptInput("Paste the authorization code:", "4/0Axx...")
    if err != nil {
        return nil, fmt.Errorf("input error: %w", err)
    }
    if code == "" {
        return nil, fmt.Errorf("no code provided")
    }

    code = extractCode(code)

    token, err := config.Exchange(context.Background(), code)
    if err != nil {
        return nil, fmt.Errorf("token exchange failed: %w", err)
    }

    if err := SaveToken(token); err != nil {
        return nil, err
    }
    return token, nil
}

func extractCode(input string) string {
    if !strings.Contains(input, "code=") {
        return input
    }
    parts := strings.SplitN(input, "?", 2)
    if len(parts) < 2 {
        return input
    }
    for _, param := range strings.Split(parts[1], "&") {
        kv := strings.SplitN(param, "=", 2)
        if len(kv) == 2 && kv[0] == "code" {
            return kv[1]
        }
    }
    return input
}

func LoadToken() (*oauth2.Token, error) {
    tokenPath := filepath.Join(ConfigDir(), "token.json")
    data, err := os.ReadFile(tokenPath)
    if err != nil {
        return nil, fmt.Errorf("run '[APP_NAME] login' first")
    }
    var token oauth2.Token
    if err := json.Unmarshal(data, &token); err != nil {
        return nil, fmt.Errorf("corrupt token file — run '[APP_NAME] login' again")
    }
    return &token, nil
}

func NewTokenSource(config *oauth2.Config, token *oauth2.Token) oauth2.TokenSource {
    return config.TokenSource(context.Background(), token)
}

func SaveToken(token *oauth2.Token) error {
    tokenPath := filepath.Join(ConfigDir(), "token.json")
    data, err := json.MarshalIndent(token, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal token: %w", err)
    }
    if err := os.WriteFile(tokenPath, data, 0600); err != nil {
        return fmt.Errorf("failed to save token: %w", err)
    }
    return nil
}

func GetHTTPClient() (*http.Client, error) {
    config, err := LoadCredentials()
    if err != nil {
        return nil, err
    }

    token, err := LoadToken()
    if err != nil {
        return nil, err
    }

    tokenSource := NewTokenSource(config, token)

    newToken, err := tokenSource.Token()
    if err != nil {
        return nil, fmt.Errorf("token refresh failed — run '[APP_NAME] login' again")
    }
    if newToken.AccessToken != token.AccessToken {
        if err := SaveToken(newToken); err != nil {
            return nil, err
        }
    }

    client := oauth2.NewClient(context.Background(), tokenSource)
    return client, nil
}

func generateState() (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    default:
        cmd = exec.Command("xdg-open", url)
    }
    return cmd.Run()
}
```

## cmd/login.go

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/[REPO_NAME]/internal/auth"
    u "github.com/[GITHUB_USER]/[REPO_NAME]/utils"
)

var loginFlags struct {
    deviceLogin bool
    manual      bool
}

var loginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with [SERVICE_NAME]",
    Run: func(cmd *cobra.Command, args []string) {
        config, err := auth.LoadCredentials()
        if err != nil {
            u.PrintFatal("failed to load credentials", err)
        }

        mode := "default"
        if loginFlags.deviceLogin {
            mode = "device"
        } else if loginFlags.manual {
            mode = "manual"
        }

        token, err := auth.Login(config, mode)
        if err != nil {
            u.PrintFatal("login failed", err)
        }

        _ = token
        u.PrintSuccess("authenticated successfully — token saved")
    },
}

func init() {
    rootCmd.AddCommand(loginCmd)

    loginCmd.Flags().BoolVar(&loginFlags.deviceLogin, "device-login", false, "Use device code flow (for headless/SSH environments)")
    loginCmd.Flags().BoolVar(&loginFlags.manual, "manual", false, "Manually paste authorization code (last resort)")
    loginCmd.MarkFlagsMutuallyExclusive("device-login", "manual")
}
```

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your CLI tool name (for config dir and error messages) |
| `[REPO_NAME]` | Go module path segment |
| `[SERVICE_NAME]` | Service being authenticated (e.g., "Google services") |
| `[DEVICE_AUTH_URL]` | Provider's device authorization endpoint (see table below) |

### Provider Device Auth URLs

| Provider | Device Auth URL |
|----------|----------------|
| Google | `https://oauth2.googleapis.com/device/code` |
| Microsoft | `https://login.microsoftonline.com/common/oauth2/v2.0/devicecode` |
| GitHub | `https://github.com/login/device/code` |

Not all providers support device authorization (e.g., Box.com does not). For those providers, omit `loginWithDevice` and the `--device-login` flag, leaving only callback and manual modes.

### Output Tier Compliance

All login flows use the correct output tiers:

| Function | Output Tier | Purpose |
|----------|-------------|---------|
| `u.PrintInfo` | Instructions, status | "Opening browser...", "Waiting for authorization..." |
| `u.PrintGeneric` | Data | URLs, user codes |
| `u.PromptInput` | Interactive input | Manual mode code paste (reads from pipe in `--for-ai`) |
| `u.PrintFatal` | Fatal errors | At command layer only (`cmd/login.go`) |
| `u.PrintSuccess` | Completion | At command layer only (`cmd/login.go`) |

### Security

- CSRF state token (16-byte random hex) is **validated on the callback flow** — the localhost server compares the redirect's `state` to the generated value and rejects a mismatch. The **manual flow** transfers the code out-of-band by hand, so there is no automated redirect to validate against; it still sends a `state` param (providers may require it) but that param is not a CSRF control there. Device flow does not use a `state` param.
- Token files stored with `0600` permissions (owner read/write only)
- Config directory created with `0700` permissions
- `openBrowser` returns error for fast-fail on headless (no 5-minute hang)
- `AccessTypeOffline` requests refresh tokens for persistent sessions

### Token Lifecycle

```
First use:  login command → Login() → save token.json
Subsequent: GetHTTPClient() → load token → auto-refresh if expired → save if refreshed
Expired:    GetHTTPClient() returns error → user runs login again
```
