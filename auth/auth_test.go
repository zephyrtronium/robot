package auth_test

import (
	"testing"
	"time"

	"github.com/zephyrtronium/robot/auth"
	"golang.org/x/oauth2"
)

func TestEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b *oauth2.Token
		want bool
	}{
		{
			name: "nils",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "right-nil",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b:    nil,
			want: false,
		},
		{
			name: "left-nil",
			a:    nil,
			b: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			want: false,
		},
		{
			name: "same",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			want: true,
		},
		{
			name: "access",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b: &oauth2.Token{
				AccessToken:  "nijika",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			want: false,
		},
		{
			name: "type",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "nijika",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			want: false,
		},
		{
			name: "refresh",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "nijika",
				Expiry:       time.Unix(0, 0),
			},
			want: false,
		},
		{
			name: "expiry",
			a: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 0),
			},
			b: &oauth2.Token{
				AccessToken:  "bocchi",
				TokenType:    "bearer",
				RefreshToken: "ryou",
				Expiry:       time.Unix(0, 1),
			},
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := auth.Equal(c.a, c.b)
			if c.want != got {
				t.Errorf("want %t got %t", c.want, got)
			}
		})
	}
}
