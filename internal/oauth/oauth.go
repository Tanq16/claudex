package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	AuthorizeURL = "https://platform.claude.com/oauth/authorize"
	TokenURL     = "https://platform.claude.com/v1/oauth/token"
	Scope        = "user:inference"

	DefaultPort      = 54545
	DefaultExpiresIn = 3600 // 1 hour
)

type Config struct {
	Port      int
	ExpiresIn int
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// urlBase64 encodes bytes as URL-safe base64 without padding.
func urlBase64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// generatePKCE returns a code_verifier and its S256 code_challenge.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generating random bytes: %w", err)
	}
	verifier = urlBase64(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = urlBase64(h[:])
	return verifier, challenge, nil
}

// generateState returns a random state parameter.
func generateState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return urlBase64(buf), nil
}

// BuildAuthorizeURL constructs the full OAuth authorize URL.
func BuildAuthorizeURL(redirectURI, challenge, state string) string {
	params := url.Values{
		"code":                  {"true"},
		"client_id":             {ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {Scope},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {state},
	}
	return AuthorizeURL + "?" + params.Encode()
}

// WaitForCallback starts a local HTTP server and waits for the OAuth callback.
// Returns the authorization code. Respects context cancellation for clean shutdown.
func WaitForCallback(ctx context.Context, port int, expectedState string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			errCh <- fmt.Errorf("authorization failed: %s", errMsg)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h2>Authentication successful</h2><p>You can close this tab.</p>")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", fmt.Errorf("starting callback server on port %d: %w", port, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()
	defer server.Shutdown(context.Background())

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("timed out waiting for authentication (5m)")
	}
}

// ExchangeCode exchanges an authorization code for an access token.
func ExchangeCode(code, verifier, state, redirectURI string, expiresIn int) (string, error) {
	payload := map[string]any{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  redirectURI,
		"client_id":     ClientID,
		"code_verifier": verifier,
		"state":         state,
		"expires_in":    expiresIn,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling token request: %w", err)
	}

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	if tokenResp.Error != "" {
		desc := tokenResp.ErrorDesc
		if desc == "" {
			desc = tokenResp.Error
		}
		return "", fmt.Errorf("token exchange failed: %s", desc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// RunFlow executes the full OAuth PKCE flow and returns an access token.
// The openBrowser callback is called with the authorize URL.
func RunFlow(ctx context.Context, cfg Config, openBrowser func(string) error) (string, error) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return "", err
	}

	state, err := generateState()
	if err != nil {
		return "", err
	}

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", cfg.Port)
	authURL := BuildAuthorizeURL(redirectURI, challenge, state)

	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("opening browser: %w", err)
	}

	code, err := WaitForCallback(ctx, cfg.Port, state)
	if err != nil {
		return "", err
	}

	return ExchangeCode(code, verifier, state, redirectURI, cfg.ExpiresIn)
}
