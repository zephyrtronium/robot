package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// dcf is a token source that uses the device code flow for initial tokens.
type dcf struct {
	mu sync.Mutex

	cfg    oauth2.Config
	st     Storage
	client *http.Client
	prompt DeviceCodePrompt
}

type DeviceCodePrompt func(userCode, verURI, verURIComplete string)

// DeviceCodeFlow creates a TokenSource which retrieves tokens through the
// device code flow. If client is nil, [http.DefaultClient] is used instead.
// prompt must be a function which prompts to navigate to the verification URI
// and enter the user code. It may be called concurrently at any time when a
// new refresh token is required.
func DeviceCodeFlow(cfg oauth2.Config, st Storage, client *http.Client, prompt DeviceCodePrompt) TokenSource {
	if cfg.Endpoint.DeviceAuthURL == "" {
		panic("auth: device code flow without device auth url")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &dcf{
		cfg:    cfg,
		st:     st,
		client: client,
		prompt: prompt,
	}
}

func (s *dcf) Token(ctx context.Context) (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, err := s.st.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve current token: %w", err)
	}
	if tok == nil {
		return s.flowLocked(ctx)
	}
	if !tok.Valid() {
		if tok.RefreshToken == "" {
			return s.flowLocked(ctx)
		}
		return s.refreshLocked(ctx, tok.RefreshToken)
	}
	return tok, nil
}

func (s *dcf) Refresh(ctx context.Context, old *oauth2.Token) (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, err := s.st.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't retrieve current token for refresh: %w", err)
	}
	if tok != nil {
		if !Equal(tok, old) {
			slog.InfoContext(ctx, "token not current, won't refresh", slog.Any("tok", tok), slog.Any("old", old))
			return tok, nil
		}
		tok, err := s.refreshLocked(ctx, tok.RefreshToken)
		switch {
		case err == nil:
			return tok, nil
		case errors.Is(err, errInvalidRefresh):
			return s.flowLocked(ctx)
		default:
			return nil, err
		}
	}
	return s.flowLocked(ctx)
}

func (s *dcf) refreshLocked(ctx context.Context, rt string) (*oauth2.Token, error) {
	// x/oauth2 doesn't expose anything to do token refresh, so we implement
	// that manually here.
	v := url.Values{
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {rt},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.Endpoint.TokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("couldn't create token refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("couldn't read refresh response body: %w", err)
	}
	var d struct {
		oauth2.Token
		Message string `json:"message"`
		Status  int    `json:"status"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("couldn't decode token refresh response: %w", err)
	}
	slog.InfoContext(ctx, "refresh response", slog.Any("unmarshaled", d), slog.String("raw", string(body)))
	if resp.StatusCode == 400 && d.Message == "Invalid refresh token" {
		return nil, fmt.Errorf("refresh failed: %w", errInvalidRefresh)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %s (%s)", d.Message, resp.Status)
	}
	tok := d.Token
	if err := s.st.Store(ctx, &tok); err != nil {
		return nil, fmt.Errorf("failed to store new token: %w", err)
	}
	return &tok, nil
}

func (s *dcf) flowLocked(ctx context.Context) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.client)
	var opts []oauth2.AuthCodeOption
	if len(s.cfg.Scopes) != 0 {
		// Twitch wants the non-standard name "scopes" rather than "scope".
		// Provide both to make everyone happy.
		opts = append(opts, oauth2.SetAuthURLParam("scopes", strings.Join(s.cfg.Scopes, " ")))
	}
	resp, err := s.cfg.DeviceAuth(ctx, opts...)
	if err != nil {
		return nil, err
	}
	s.prompt(resp.UserCode, resp.VerificationURI, resp.VerificationURIComplete)
	var tok *oauth2.Token
	for {
		tok, err = s.cfg.DeviceAccessToken(ctx, resp)
		if err == nil {
			break
		}
		if isPending(err) {
			continue
		}
		return nil, fmt.Errorf("failed to get token from device code flow: %w", err)
	}
	if err := s.st.Store(ctx, tok); err != nil {
		return nil, fmt.Errorf("failed to store first token: %w", err)
	}
	return tok, nil
}

// isPending returns whether the error indicates that device code authorization
// is pending the user's input.
func isPending(err error) bool {
	r := new(oauth2.RetrieveError)
	if !errors.As(err, &r) {
		return false
	}
	var v struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(r.Body, &v); err != nil {
		return false
	}
	return v.Status == 400 && v.Message == "authorization_pending"
}

var errInvalidRefresh = errors.New("invalid refresh token")
