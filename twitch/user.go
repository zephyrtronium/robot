package twitch

import (
	"context"
	"fmt"
	"net/url"
)

// User is the response type from https://dev.twitch.tv/docs/api/reference/#get-users.
type User struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	Email           string `json:"email"`
	CreatedAt       string `json:"created_at"`
}

func UsersByLogin(ctx context.Context, client Client, names ...string) ([]User, error) {
	v := url.Values{"login": names}
	u := make([]User, 0, 100)
	url := apiurl("/helix/users", v)
	err := reqjson(ctx, client, "GET", url, nil, &u)
	if err != nil {
		return nil, fmt.Errorf("couldn't get users info: %w", err)
	}
	return u, nil
}

func UsersByID(ctx context.Context, client Client, ids ...string) ([]User, error) {
	v := url.Values{"id": ids}
	u := make([]User, 0, 100)
	url := apiurl("/helix/users", v)
	err := reqjson(ctx, client, "GET", url, nil, &u)
	if err != nil {
		return nil, fmt.Errorf("couldn't get users info: %w", err)
	}
	return u, nil
}
