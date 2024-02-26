package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type memStorage struct {
	v *oauth2.Token
}

// Load returns the current refresh token. If the result is nil,
// the caller should use acquire a new refresh token.
func (s *memStorage) Load(ctx context.Context) (*oauth2.Token, error) {
	return s.v, nil
}

// Store sets a new token. If tok nil, the storage should be cleared.
func (s *memStorage) Store(ctx context.Context, tok *oauth2.Token) error {
	s.v = tok
	return nil
}

type fixedResponse struct {
	status int
	body   string
}

func (f *fixedResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(f.status)
	io.WriteString(w, f.body)
}

func dcfFixture(t *testing.T) (prompts *int, src *dcf) {
	t.Helper()
	var mux http.ServeMux
	deviceResp := &fixedResponse{
		status: 200,
		body:   `{"device_code":"bocchi","expires_in":1800,"interval":1,"user_code":"ryou","verification_uri":"http://error/"}`,
	}
	tokenResp := &fixedResponse{
		status: 200,
		body:   `{"access_token":"nijika","expires_in":14400,"refresh_token":"kita","scope":[],"token_type":"bearer"}`,
	}
	mux.Handle("POST /oauth2/device", deviceResp)
	mux.Handle("POST /oauth2/token", tokenResp)
	srv := httptest.NewServer(&mux)
	t.Cleanup(srv.Close)
	deviceURL, err := url.JoinPath(srv.URL, "/oauth2/device")
	if err != nil {
		panic(err)
	}
	tokenURL, err := url.JoinPath(srv.URL, "/oauth2/token")
	if err != nil {
		panic(err)
	}
	cfg := oauth2.Config{
		ClientID:     "bocchi",
		ClientSecret: "ryou",
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: deviceURL,
			TokenURL:      tokenURL,
		},
		Scopes: []string{"chat:read", "chat:edit"},
	}
	st := new(memStorage)
	client := &http.Client{Timeout: 30 * time.Second}
	prompts = new(int)
	prompt := func(string, string, string) { *prompts++ }
	src = DeviceCodeFlow(cfg, st, client, prompt).(*dcf)
	return prompts, src
}

func TestDCFItself(t *testing.T) {
	t.Parallel()
	prompts, src := dcfFixture(t)
	// It's flowLocked, but there's no chance of concurrency here.
	tok, err := src.flowLocked(context.Background())
	if err != nil {
		t.Fatalf("couldn't get token: %v", err)
	}
	if tok.AccessToken != "nijika" {
		t.Errorf("wrong access token: want %q, got %q", "nijika", tok.AccessToken)
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts: want 1, got %d", *prompts)
	}
}

func TestDCFRefreshReq(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prompts, src := dcfFixture(t)
	first, err := src.refreshLocked(ctx, "ryou")
	if err != nil {
		t.Fatalf("couldn't perform first token refresh: %v", err)
	}
	if first == nil {
		t.Fatal("first token is nil")
	}
	snd, err := src.refreshLocked(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("couldn't perform second token refresh: %v", err)
	}
	if snd == first {
		t.Error("second token refresh didn't make a new token")
	}
	if *prompts != 0 {
		t.Errorf("wrong number of prompts after refresh: want 0, got %d", *prompts)
	}
}

func TestDCFRefresh(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prompts, src := dcfFixture(t)
	first, err := src.Refresh(ctx, nil)
	if err != nil {
		t.Errorf("couldn't perform first token refresh: %v", err)
	}
	if first == nil {
		t.Error("first token is nil")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after first refresh: want 1, got %d", *prompts)
	}
	swap, err := src.Refresh(ctx, first)
	if err != nil {
		t.Errorf("couldn't perform swap token refresh: %v", err)
	}
	if swap == first {
		t.Error("swap didn't swap")
	}
	if swap == nil {
		t.Error("swap got nil")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after swap: want 1, got %d", *prompts)
	}
	swapnt, err := src.Refresh(ctx, first)
	if err != nil {
		t.Errorf("couldn't perform swapn't token refresh: %v", err)
	}
	if swapnt != swap {
		t.Error("swapn't swapped")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after swapn't: want 1, got %d", *prompts)
	}
}

func TestDCFToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prompts, src := dcfFixture(t)
	first, err := src.Token(ctx)
	if err != nil {
		t.Errorf("couldn't get first token: %v", err)
	}
	if first == nil {
		t.Error("first token is nil")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after first token: want 1, got %d", *prompts)
	}
	snd, err := src.Token(ctx)
	if err != nil {
		t.Errorf("couldn't get second token: %v", err)
	}
	if snd != first {
		t.Error("second token was different from first")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after second token: want 1, got %d", *prompts)
	}
	// Force a refresh.
	if snd == nil {
		return
	}
	snd.Expiry = time.Now().Add(-time.Hour)
	third, err := src.Token(ctx)
	if err != nil {
		t.Errorf("couldn't get token after expiration: %v", err)
	}
	if third == snd {
		t.Error("expired token didn't refresh")
	}
	if third == nil {
		t.Error("expired token became nil")
	}
	if *prompts != 1 {
		t.Errorf("wrong number of prompts after expired token: want 1, got %d", *prompts)
	}
}
