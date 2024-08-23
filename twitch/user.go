package twitch

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
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

// Users gets user information for a list of up to 100 users.
// For each user in the given list, if the ID is provided, then the query is
// made by ID, and otherwise it is made by login.
// Logins used to search are normalized to lower case.
// The result reuses the memory in users, but may be of different length.
//
// If a given user has both an ID and a login, the ID is used and the login
// is replaced with the result from the API.
// If a user has neither, it is ignored.
func Users(ctx context.Context, client Client, tok *oauth2.Token, users []User) ([]User, error) {
	v := url.Values{
		"id":    nil,
		"login": nil,
	}
	for _, u := range users {
		if u.ID != "" {
			v["id"] = append(v["id"], u.ID)
			continue
		}
		if u.Login != "" {
			v["login"] = append(v["login"], strings.ToLower(u.Login))
		}
	}
	url := apiurl("/helix/users", v)
	err := reqjson(ctx, client, tok, "GET", url, nil, &users)
	if err != nil {
		return nil, fmt.Errorf("couldn't get users info: %w", err)
	}
	return users, nil
}
