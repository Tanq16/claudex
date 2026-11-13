---
name: go-oauth-cli
description: OAuth2 login for a Go CLI client - one flow that prints the authorization URL and reads back the redirect address the user pastes, a token cached under ~/.config, and refresh on every use. Use when a CLI tool authenticates against Google, GitHub, Microsoft, or another OAuth2 provider, when building internal/auth/, or when adding a login command. Triggers on golang.org/x/oauth2, AuthCodeURL, config.Exchange, TokenSource, token.json, credentials.json, and cmd/login.go. Not for server-side web OAuth where the browser stays on your own site.
user-invocable: false
---

# Go OAuth for CLI Clients

**One login flow that prints a URL and reads one pasted line back, a token cached at `~/.config/[APP_NAME]/token.json`, and automatic refresh on every use.**

This covers a terminal tool obtaining a token for itself. A web server authenticating its own visitors is a different problem with a different session model.

## The Flow

`login` prints the authorization URL and reads one line. The redirect goes to `http://localhost`, where nothing listens, so the browser lands on a connection error with the code in its address bar, and the user pastes that address back. The same flow works on a desktop, over SSH, and in a container, so there is no mode to pick and no flag to pick it with.

The pasted line has no flag behind it. The code does not exist until the URL has been printed, so no invocation could carry it in advance.

| Call | Used for |
|---|---|
| `u.PrintInfo` | the instruction above the URL |
| `u.PrintGeneric` | the URL |
| `u.PromptInput` | the pasted redirect address |
| `u.PrintFatal`, `u.PrintSuccess` | the command layer only |

The URL goes through `PrintGeneric` so it arrives unstyled and unprefixed, which keeps it copyable out of a terminal or a pipe.

## Security

The token file is written at `0600` and the config directory created at `0700`, since a bearer token in a shared home directory is a credential anyone on the box can read.

`AccessTypeOffline` requests a refresh token, without which the user re-authenticates every hour.

## internal/auth/auth.go

```go
package auth

import (
    "context"
    "crypto/rand"
    "encoding/json/jsontext"
    "encoding/json/v2"
    "errors"
    "fmt"
    "net/http"
    "net/url"
    "os"
    "path/filepath"

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
    // Google: google.ConfigFromJSON(data, scopes...)
    // Others: construct oauth2.Config directly with the provider's endpoints
    return config, nil
}

func Login(config *oauth2.Config) (*oauth2.Token, error) {
    config.RedirectURL = "http://localhost"
    authURL := config.AuthCodeURL(rand.Text(), oauth2.AccessTypeOffline, oauth2.ApprovalForce)

    u.PrintInfo("Open this URL, approve access, then copy the address the browser lands on:")
    u.PrintGeneric(authURL)

    pasted, err := u.PromptInput("Paste the address:")
    if err != nil {
        return nil, fmt.Errorf("reading the redirect address: %w", err)
    }
    code := extractCode(pasted)
    if code == "" {
        return nil, errors.New("no authorization code pasted")
    }

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
    if parsed, err := url.Parse(input); err == nil && parsed.Query().Has("code") {
        return parsed.Query().Get("code")
    }
    return input
}
```

`extractCode` accepts the full redirect address as well as a bare code, because the address bar is what the user can copy, and isolating one query parameter by hand invites a truncated paste.

`AuthCodeURL` takes a state value, and `rand.Text` supplies a random one in a single call. Nothing redirects back automatically for the tool to compare it against, so it is not a CSRF control in this flow.

```go
func LoadToken() (*oauth2.Token, error) {
    data, err := os.ReadFile(filepath.Join(ConfigDir(), "token.json"))
    if err != nil {
        return nil, fmt.Errorf("run '[APP_NAME] login' first")
    }
    var token oauth2.Token
    if err := json.Unmarshal(data, &token); err != nil {
        return nil, fmt.Errorf("corrupt token file, run '[APP_NAME] login' again")
    }
    return &token, nil
}

func SaveToken(token *oauth2.Token) error {
    data, err := json.Marshal(token, jsontext.WithIndent("  "))
    if err != nil {
        return fmt.Errorf("failed to marshal token: %w", err)
    }
    if err := os.WriteFile(filepath.Join(ConfigDir(), "token.json"), data, 0600); err != nil {
        return fmt.Errorf("failed to save token: %w", err)
    }
    return nil
}

// GetHTTPClient is the entry point every authenticated call uses.
func GetHTTPClient() (*http.Client, error) {
    config, err := LoadCredentials()
    if err != nil {
        return nil, err
    }
    token, err := LoadToken()
    if err != nil {
        return nil, err
    }

    src := config.TokenSource(context.Background(), token)
    fresh, err := src.Token()
    if err != nil {
        return nil, fmt.Errorf("token refresh failed, run '[APP_NAME] login' again")
    }
    if fresh.AccessToken != token.AccessToken {
        if err := SaveToken(fresh); err != nil {
            return nil, err
        }
    }
    return oauth2.NewClient(context.Background(), src), nil
}
```

`GetHTTPClient` writes the token back only when the refresh actually produced a new one, so a read-only run does not rewrite the file on every invocation.

## cmd/login.go

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/[GITHUB_USER]/[REPO_NAME]/internal/auth"
    u "github.com/[GITHUB_USER]/[REPO_NAME]/utils"
)

var loginCmd = &cobra.Command{
    Use:   "login",
    Short: "Authenticate with [SERVICE_NAME]",
    Args:  cobra.NoArgs,
    Run: func(cmd *cobra.Command, args []string) {
        config, err := auth.LoadCredentials()
        if err != nil {
            u.PrintFatal("failed to load credentials", err)
        }
        if _, err := auth.Login(config); err != nil {
            u.PrintFatal("login failed", err)
        }
        u.PrintSuccess("authenticated successfully, token saved")
    },
}

func init() {
    rootCmd.AddCommand(loginCmd)
}
```

## Placeholders

| Placeholder | Replace with |
|---|---|
| `[APP_NAME]` | the CLI tool's name, used for the config directory and error messages |
| `[REPO_NAME]` | the module path segment |
| `[SERVICE_NAME]` | the service being authenticated against |
