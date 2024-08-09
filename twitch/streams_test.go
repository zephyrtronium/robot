package twitch

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
)

func TestUserStreams(t *testing.T) {
	// Test that we successfully interpret the example response in the API doc.
	t.Run("decode", func(t *testing.T) {
		spy := apiresp(200, "streams.json")
		cl := Client{
			HTTP: &http.Client{
				Transport: spy,
			},
			Token: &oauth2.Token{
				AccessToken: "bocchi",
			},
		}
		s := []Stream{
			{UserID: "98765"},
			{UserLogin: "sandysanderman"},
		}
		s, err := UserStreams(context.Background(), cl, s)
		if err != nil {
			t.Error(err)
		}
		if len(s) != 1 {
			t.Fatalf("wrong number of results: want 1, got %d", len(s))
		}
		want := Stream{
			ID:           "40952121085",
			UserID:       "101051819",
			UserLogin:    "afro",
			UserName:     "Afro",
			GameID:       "32982",
			GameName:     "Grand Theft Auto V",
			Type:         "live",
			Title:        "Jacob: Digital Den Laptops & Routers | NoPixel | !MAINGEAR !FCF",
			Tags:         []string{"English"},
			ViewerCount:  1490,
			StartedAt:    time.Date(2021, 3, 10, 3, 18, 11, 0, time.UTC),
			Language:     "en",
			ThumbnailURL: "https://static-cdn.jtvnw.net/previews-ttv/live_user_afro-{width}x{height}.jpg",
			IsMature:     false,
		}
		got := s[0]
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("wrong result (+got/-want):\n%s", diff)
		}
	})
}
