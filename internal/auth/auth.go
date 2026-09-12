package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	u "github.com/tanq16/claudex/utils"
)

const (
	ClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	AuthorizeURL = "https://platform.claude.com/oauth/authorize"
	TokenURL     = "https://platform.claude.com/v1/oauth/token"
	Scope        = "user:inference"

	DefaultExpiresIn = 3600

	manualRedirectURI = "http://localhost/callback"
	callbackTimeout   = 5 * time.Minute
)

type Config struct {
	Port      int
	ExpiresIn int
	Manual    bool
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

func urlBase64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

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

func generateState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return urlBase64(buf), nil
}

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

func Login(ctx context.Context, cfg Config, openBrowser func(string) error) (string, error) {
	verifier, challenge, err := generatePKCE()
	if err != nil {
		return "", err
	}
	state, err := generateState()
	if err != nil {
		return "", err
	}
	if cfg.Manual {
		return loginManual(ctx, cfg, verifier, challenge, state)
	}
	return loginCallback(ctx, cfg, openBrowser, verifier, challenge, state)
}

func loginCallback(ctx context.Context, cfg Config, openBrowser func(string) error, verifier, challenge, state string) (string, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		return "", fmt.Errorf("starting callback server: %w", err)
	}
	defer listener.Close()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", listener.Addr().(*net.TCPAddr).Port)
	authURL := BuildAuthorizeURL(redirectURI, challenge, state)

	announce("Opening browser for authentication...")
	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("opening browser, retry with --manual: %w", err)
	}
	announce("Waiting for authorization in browser...")
	announceURL(authURL)

	code, err := waitForCallback(ctx, listener, state)
	if err != nil {
		return "", err
	}
	return exchangeCode(ctx, code, verifier, state, redirectURI, cfg.ExpiresIn)
}

func loginManual(ctx context.Context, cfg Config, verifier, challenge, state string) (string, error) {
	authURL := BuildAuthorizeURL(manualRedirectURI, challenge, state)

	u.PrintInfo("Visit this URL to authenticate:")
	u.PrintGeneric(authURL)
	u.PrintInfo("After authorizing, copy the 'code' parameter from the redirect URL.")

	pasted, err := u.PromptInput("Paste the authorization code:", "")
	if err != nil {
		return "", err
	}
	if pasted == "" {
		return "", errors.New("no authorization code provided")
	}
	return exchangeCode(ctx, extractCode(pasted), verifier, state, manualRedirectURI, cfg.ExpiresIn)
}

func extractCode(input string) string {
	if !strings.Contains(input, "code=") {
		return input
	}
	_, query, found := strings.Cut(input, "?")
	if !found {
		return input
	}
	for param := range strings.SplitSeq(query, "&") {
		if k, v, ok := strings.Cut(param, "="); ok && k == "code" {
			return v
		}
	}
	return input
}

func waitForCallback(ctx context.Context, listener net.Listener, expectedState string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Non-blocking sends so a duplicate /callback request can never park a handler goroutine on a full channel.
	sendCode := func(code string) {
		select {
		case codeCh <- code:
		default:
		}
	}
	sendErr := func(err error) {
		select {
		case errCh <- err:
		default:
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			sendErr(fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state))
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			sendErr(fmt.Errorf("authorization failed: %s", errMsg))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h2>Authentication successful</h2><p>You can close this tab.</p>")
		sendCode(code)
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			sendErr(fmt.Errorf("callback server error: %w", err))
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(callbackTimeout):
		return "", fmt.Errorf("timed out waiting for authentication (%s)", callbackTimeout)
	}
}

func exchangeCode(ctx context.Context, code, verifier, state, redirectURI string, expiresIn int) (string, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(string(body)))
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
	if err := json.UnmarshalRead(resp.Body, &tokenResp); err != nil {
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
		return "", errors.New("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

func announce(msg string) {
	if u.GlobalDebugFlag || u.StdoutIsTerminal {
		u.PrintInfo(msg)
	}
}

func announceURL(authURL string) {
	if u.GlobalDebugFlag || u.StdoutIsTerminal {
		u.PrintGeneric(authURL)
	}
}
