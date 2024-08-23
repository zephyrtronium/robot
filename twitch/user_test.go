package twitch

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
)

func TestUsers(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		spy := apiresp(200, "users.json")
		cl := Client{
			HTTP: &http.Client{
				Transport: spy,
			},
		}
		tok := &oauth2.Token{AccessToken: "bocchi"}
		u := []User{
			{ID: "141981764"},
			{Login: "twitchdev"},
		}
		u, err := Users(context.Background(), cl, tok, u)
		if err != nil {
			t.Error(err)
		}
		if len(u) != 1 {
			t.Fatalf("wrong number of results: want 1, got %d", len(u))
		}
		want := User{
			ID:              "141981764",
			Login:           "twitchdev",
			DisplayName:     "TwitchDev",
			Type:            "",
			BroadcasterType: "partner",
			Description:     "Supporting third-party developers building Twitch integrations from chatbots to game integrations.",
			ProfileImageURL: "https://static-cdn.jtvnw.net/jtv_user_pictures/8a6381c7-d0c0-4576-b179-38bd5ce1d6af-profile_image-300x300.png",
			OfflineImageURL: "https://static-cdn.jtvnw.net/jtv_user_pictures/3f13ab61-ec78-4fe6-8481-8682cb3b0ac2-channel_offline_image-1920x1080.png",
			ViewCount:       5980557,
			Email:           "not-real@email.com",
			CreatedAt:       "2016-12-14T20:32:28Z",
		}
		got := u[0]
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("wrong result (+got/-want):\n%s", diff)
		}
	})
}
