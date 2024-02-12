package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/oauth2"
)

// Token represents a persistent, refreshing OAuth2 credential.
type Token struct {
	// access holds the current access token.
	access atomic.Value // string
	// mu synchronizes refreshes.
	mu sync.Mutex
	// refresh is the current refresh token.
	refresh string
	// storage is the token storage.
	storage Storage
	// app is the OAuth2 flow configuration.
	app oauth2.Config
	// client is the HTTP client for token requests.
	client http.Client
	// ch is the channel over which the authorization code grant callback
	// sends the new refresh token.
	ch chan string
	// landing is the callback's own redirect destination.
	landing string
	// rand is the source of randomness for generating state parameters. It
	// should usually be crypto/rand.Reader.
	rand io.Reader
}

// Config is the configuration for acquiring tokens.
type Config struct {
	// App is the OAuth2 endpoint and client configuration.
	App oauth2.Config
	// Client is the HTTP client used for acquiring and refreshing tokens.
	Client http.Client
	// Landing is the URI to which the OAuth2 callback redirects. If it is the
	// empty string, the callback redirects to / instead.
	Landing string
}

// New creates a new token.
func New(ctx context.Context, s Storage, cfg Config) (*Token, error) {
	r, err := s.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't load token: %w", err)
	}
	// Note that even if err is nil, r may be the empty string, indicating that
	// we need to do the authorization code grant flow.
	t := Token{
		refresh: r,
		storage: s,
		app:     cfg.App,
		client:  cfg.Client,
		ch:      make(chan string, 1),
		landing: cfg.Landing,
		rand:    rand.Reader,
	}
	if t.landing == "" {
		t.landing = "/"
	}
	// Store strings in our atomic variables so we can always assert Load.
	t.access.Store("")
	return &t, nil
}

// Access obtains the current access token. The token may be invalid as
// determined by the API. If using the token results in an unauthorized status,
// the caller should call Refresh and attempt the operation again. If there is
// no access token currently stored, including if the token is being refreshed,
// Access will block until one is available.
func (t *Token) Access(ctx context.Context) (string, error) {
	r := t.access.Load().(string)
	if r == "" {
		return t.Refresh(ctx, "")
	}
	return r, nil
}

// Refresh refreshes the access token and returns its new value. cur must be
// the current (invalid) access token. If there is no refresh token, Refresh
// waits for completion of the authorization code grant flow.
func (t *Token) Refresh(ctx context.Context, cur string) (string, error) {
	t.mu.Lock()
	// Can't defer unlock because we might recurse – we need to unlock before
	// returning, not after.
	// Check the context after the wait.
	if ctx.Err() != nil {
		t.mu.Unlock()
		return "", ctx.Err()
	}
	if !t.access.CompareAndSwap(cur, "") {
		// Another call to Refresh completed while we were acquiring t.mu.
		t.mu.Unlock()
		return t.Access(ctx)
	}
	r, err := t.refreshLocked(ctx)
	t.mu.Unlock()
	return r, err
}

// refreshLocked implements the actual token refresh logic.
func (t *Token) refreshLocked(ctx context.Context) (string, error) {
	if t.refresh == "" {
		// No refresh token. Perform the authorization code grant flow. It sets
		// the new access token for us.
		err := t.authcode(ctx)
		return t.access.Load().(string), err
	}

	v := url.Values{
		"client_id":     {t.app.ClientID},
		"client_secret": {t.app.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {t.refresh},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", t.app.Endpoint.TokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return "", fmt.Errorf("couldn't create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		// TODO(zeph): we probably shouldn't just give up here. but since we
		// do, should we clear the refresh token?
		return "", fmt.Errorf("couldn't call refresh endpoint: %w", err)
	}
	defer resp.Body.Close()

	// TODO(zeph): This type definition, particularly error fields, currently
	// reflects Twitch API.
	var result struct {
		// Success fields
		AccessToken  string   `json:"access_token"`
		RefreshToken string   `json:"refresh_token"`
		Scope        []string `json:"scope"`
		TokenType    string   `json:"token_type"`
		ExpiresIn    int      `json:"expires_in"`
		// Error fields
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	d := json.NewDecoder(resp.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&result); err != nil {
		return "", fmt.Errorf("couldn't decode refresh response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("couldn't refresh access token: %s (%d %s)", result.Message, result.Status, result.Error)
	}
	if err := t.storage.Store(ctx, result.RefreshToken); err != nil {
		return result.AccessToken, fmt.Errorf("couldn't store new refresh token: %w", err)
	}
	return result.AccessToken, nil
}

// authcode waits for completion of the authorization code grant flow.
func (t *Token) authcode(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-t.ch:
		t.refresh = r
		err := t.storage.Store(ctx, r)
		if err != nil {
			return fmt.Errorf("couldn't save new refresh token: %w", err)
		}
		return nil
	}
}

// cookieName returns the name of the cookie used to store the OAuth2 state.
func (t *Token) cookieName() string {
	return "oauth2_state_" + t.app.ClientID
}

// Login is an HTTP endpoint that redirects to the token's configured OAuth2
// authorization code grant URL if the client is awaiting authorization.
func (t *Token) Login(w http.ResponseWriter, r *http.Request) {
	state := newstate(t.rand)
	// NOTE(zeph): We could AEAD the state to prevent requests with any
	// matching query parameter and cookie value from "passing" the CSRF
	// challenge, but the callback is idempotent up to success, and it will
	// simply fail to exchange an invalid authorization code.
	c := http.Cookie{
		Name:  t.cookieName(),
		Value: state,
		Path:  "/",
		// We don't set Secure because we generally expect to serve HTTP over
		// an isolated network rather than serving HTTPS.
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &c)
	http.Redirect(w, r, t.app.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

// Callback is an HTTP endpoint that serves the token's OAuth2 authorization
// code grant callback. It must be served at the token's configured redirect
// URI.
func (t *Token) Callback(w http.ResponseWriter, r *http.Request) {
	if err := t.checkstate(r); err != nil {
		http.Error(w, "An error occurred: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, t.client)
	code := r.FormValue("code")
	token, err := t.app.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Authorization code exchange failed: "+err.Error(), http.StatusBadRequest)
	}
	// Another goroutine is waiting in t.authcode for us to send the refresh
	// token, but we can make the access token available now.
	t.access.Store(token.AccessToken)
	go func(r string) {
		// If there's an existing refresh token, yoink it out so we're
		// providing the most recent one.
		select {
		case <-t.ch:
		default:
		}
		t.ch <- r
	}(token.RefreshToken)

	nc := http.Cookie{
		Name:     t.cookieName(),
		Path:     "/",
		MaxAge:   -1, // delete immediately
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &nc)
	http.Redirect(w, r, t.landing, http.StatusSeeOther)
}

func (t *Token) checkstate(r *http.Request) error {
	c, err := r.Cookie(t.cookieName())
	if err == http.ErrNoCookie || c.Value == "" {
		return errors.New("client is not currently expecting an authorization code")
	}
	if err != nil {
		return err
	}
	want := c.Value
	got := r.FormValue("state")
	if want != got {
		return errors.New("CSRF challenge failed")
	}
	return nil
}

func newstate(r io.Reader) string {
	b := make([]byte, 12)
	if _, err := io.ReadFull(r, b); err != nil {
		panic(fmt.Errorf("couldn't read 12 bytes from %T state: %w", r, err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
