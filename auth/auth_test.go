package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

type testStorage struct {
	v string
}

func (t *testStorage) Load(ctx context.Context) (string, error) {
	return t.v, nil
}

func (t *testStorage) Store(ctx context.Context, rt string) error {
	t.v = rt
	return nil
}

func TestAuth(t *testing.T) {
	cases := []struct {
		name   string
		ep     testEndpoint
		client string
		secret string
		// TODO(zeph): test that we get an error with wrong client/secret
	}{
		{
			name: "ok",
			ep: testEndpoint{
				code:    "bocchi",
				client:  "kita",
				secret:  "nijika",
				access:  "ryou",
				refresh: "hiroi",
			},
			client: "kita",
			secret: "nijika",
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			tok, url := startServers(t, &c.ep, c.client, c.secret)
			ch := make(chan struct{})
			go login(ctx, ch, url)
			<-ch
			access, err := tok.Access(ctx)
			if err != nil {
				t.Errorf("failed to get access token: %v", err)
			}
			if access != c.ep.access {
				t.Errorf("wrong access token: want %q, got %q", c.ep.access, access)
			}
			tok.mu.Lock()
			refresh := tok.refresh
			tok.mu.Unlock()
			if refresh != c.ep.refresh {
				t.Errorf("wrong refresh token: want %q, got %q", c.ep.refresh, refresh)
			}
			hits := c.ep.tokens.Load()
			if c.ep.authorizes.Load() != 1 {
				t.Errorf("wrong number of authorization calls: want 1, got %d", c.ep.authorizes.Load())
			}
			// Calling Access again should not hit endpoints.
			access, _ = tok.Access(ctx)
			if c.ep.authorizes.Load() != 1 || c.ep.tokens.Load() != hits {
				t.Errorf("wrong number of endpoint calls after second access: want 1/%d, got %d/%d", hits, c.ep.authorizes.Load(), c.ep.tokens.Load())
			}
			// Refreshing should always go through.
			_, err = tok.Refresh(ctx, access)
			if err != nil {
				t.Errorf("refresh failed: %v", err)
			}
			if c.ep.authorizes.Load() != 1 || c.ep.tokens.Load() != hits+1 {
				t.Errorf("wrong number of endpoint calls after refresh: want 1/%d, got %d/%d", hits+1, c.ep.authorizes.Load(), c.ep.tokens.Load())
			}
		})
	}
}

func TestAccessConcurrent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: connectex: No connection could be made because the target machine actively refused it.")
	}
	grp, ctx := errgroup.WithContext(context.Background())
	ep := testEndpoint{
		code:    "bocchi",
		client:  "kita",
		secret:  "nijika",
		access:  "ryou",
		refresh: "hiroi",
	}
	tok, url := startServers(t, &ep, ep.client, ep.secret)
	ch := make(chan struct{})
	for i := 0; i < 1000; i++ {
		grp.Go(func() error {
			<-ch
			access, err := tok.Access(ctx)
			if err != nil {
				return fmt.Errorf("couldn't get access token: %w", err)
			}
			if access != ep.access {
				return fmt.Errorf("wrong access token: want %q, got %q", ep.access, access)
			}
			return nil
		})
	}
	go login(context.Background(), ch, url)
	if err := grp.Wait(); err != nil {
		t.Error(err)
	}
	if ep.authorizes.Load() != 1 || ep.tokens.Load() > 2 {
		t.Errorf("wrong number of endpoint hits: want 1/1 or 1/2, got %d/%d", ep.authorizes.Load(), ep.tokens.Load())
	}
}

func TestRefreshConcurrent(t *testing.T) {
	grp, ctx := errgroup.WithContext(context.Background())
	ep := testEndpoint{
		code:    "bocchi",
		client:  "kita",
		secret:  "nijika",
		access:  "ryou",
		refresh: "hiroi",
	}
	tok, url := startServers(t, &ep, ep.client, ep.secret)
	ch := make(chan struct{})
	go login(context.Background(), ch, url)
	start, err := tok.Refresh(context.Background(), "")
	if err != nil {
		t.Fatalf("couldn't perform initial refresh: %v", err)
	}
	for i := 0; i < 1000; i++ {
		grp.Go(func() error {
			<-ch
			// NOTE(zeph): This only works because our test endpoint returns
			// the same tokens every time.
			access, err := tok.Refresh(ctx, start)
			if err != nil {
				return fmt.Errorf("couldn't get access token: %w", err)
			}
			if access != ep.access {
				return fmt.Errorf("wrong access token: want %q, got %q", ep.access, access)
			}
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		t.Error(err)
	}
	if ep.authorizes.Load() != 1 || ep.tokens.Load() < 1000 {
		t.Errorf("wrong number of endpoint hits: want 1/1000+, got %d/%d", ep.authorizes.Load(), ep.tokens.Load())
	}
}

func login(ctx context.Context, ch chan<- struct{}, url string) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	cl := http.Client{Jar: jar}
	req, err := http.NewRequestWithContext(ctx, "GET", url+"/login", nil)
	if err != nil {
		panic(err)
	}
	close(ch)
	if _, err := cl.Do(req); err != nil {
		panic(err)
	}
}

func startServers(t *testing.T, tep *testEndpoint, client, secret string) (tok *Token, url string) {
	t.Helper()
	rmt, ep := tep.server()
	t.Cleanup(rmt.Close)
	app := oauth2.Config{
		ClientID:     client,
		ClientSecret: secret,
		Endpoint:     ep,
		RedirectURL:  "/callback",
		Scopes:       []string{"chat:read", "chat:edit"},
	}
	tok, err := New(context.Background(), new(testStorage), Config{App: app})
	if err != nil {
		t.Fatal(err)
	}
	srv := server(tok)
	t.Cleanup(srv.Close)
	return tok, srv.URL
}

func server(tok *Token) *httptest.Server {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Get("/login", tok.Login)
	r.Get("/callback", tok.Callback)
	s := httptest.NewServer(r)
	tok.app.RedirectURL = s.URL + tok.app.RedirectURL
	return s
}
