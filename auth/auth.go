package auth

import (
	"context"

	"golang.org/x/oauth2"
)

// TokenSource is a source of OAuth2 access tokens. Its methods are safe to
// call concurrently.
type TokenSource interface {
	// Token retrieves a token value. This may trigger OAuth2 flows including
	// token refresh, device code flow, or authorization code grant flow.
	// The result is always non-nil if the error is nil.
	Token(ctx context.Context) (*oauth2.Token, error)
	// Refresh forces a refresh of the token if its current value is identical
	// to old in the sense of [Equal]. This may trigger OAuth2 flows.
	// The result is the refreshed token.
	// The requirement to provide the old token allows Refresh to be called
	// concurrently without flooding refresh requests.
	Refresh(ctx context.Context, old *oauth2.Token) (*oauth2.Token, error)
}

// Equal compares two OAuth2 tokens by access token, refresh token, token type,
// and expiry.
func Equal(a, b *oauth2.Token) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	return a.AccessToken == b.AccessToken &&
		a.TokenType == b.TokenType &&
		a.RefreshToken == b.RefreshToken &&
		a.Expiry.Equal(b.Expiry)
}
