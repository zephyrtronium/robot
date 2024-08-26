package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

type ccf struct {
	mu  sync.Mutex
	cur *oauth2.Token

	cfg    oauth2.Config
	client *http.Client
}

// ClientCredentialsFlow creates a TokenSource which retrieves tokens through
// the client credentials grant flow.
// If client is nil, [http.DefaultClient] is used instead.
// Note that the client credentials flow does not have refresh tokens, so the
// tokens are not stored across processes.
func ClientCredentialsFlow(cfg oauth2.Config, client *http.Client) TokenSource {
	if client == nil {
		client = http.DefaultClient
	}
	return &ccf{
		cfg:    cfg,
		client: client,
	}
}

// Token retrieves a token value.
// The result is always non-nil if the error is nil.
func (c *ccf) Token(ctx context.Context) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cur != nil {
		return c.cur, nil
	}
	return c.flowLocked(ctx)
}

// Refresh forces a refresh of the token if its current value is identical
// to old in the sense of [Equal].
// The result is the refreshed token.
// The requirement to provide the old token allows Refresh to be called
// concurrently without flooding refresh requests.
func (c *ccf) Refresh(ctx context.Context, old *oauth2.Token) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !Equal(c.cur, old) {
		return c.cur, nil
	}
	return c.flowLocked(ctx)
}

func (c *ccf) flowLocked(ctx context.Context) (*oauth2.Token, error) {
	// Client credentials flow is unusual, so it seems we get to hand-roll it.
	v := url.Values{
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"grant_type":    {"client_credentials"},
	}
	slog.LogAttrs(ctx, slog.LevelDebug-4, "ccf refresh ### THIS MESSAGE CONTAINS SECRETS ###", slog.Any("values", v))
	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.Endpoint.TokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("couldn't create client credentials request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client credentials request failed: %w", err)
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
	slog.InfoContext(ctx, "client credentials", slog.Int("status", d.Status))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %s (%s)", d.Message, resp.Status)
	}
	c.cur = &d.Token
	return c.cur, nil
}
