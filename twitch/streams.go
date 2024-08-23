package twitch

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// Stream is the response type from https://dev.twitch.tv/docs/api/reference/#get-streams.
type Stream struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	UserLogin    string    `json:"user_login"`
	UserName     string    `json:"user_name"`
	GameID       string    `json:"game_id"`
	GameName     string    `json:"game_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Tags         []string  `json:"tags"`
	ViewerCount  int       `json:"viewer_count"`
	StartedAt    time.Time `json:"started_at"`
	Language     string    `json:"language"`
	ThumbnailURL string    `json:"thumbnail_url"`
	IsMature     bool      `json:"is_mature"`
	// tag_ids is deprecated and always empty
}

// UserStreams gets stream information for a list of up to 100 users.
// For each stream in the given list, if the user ID is provided, then the
// query is made by user ID, and otherwise it is made by user login.
// (NOTE: the ID field is not used as input; only UserID.)
// Logins used to search are normalized to lower case.
// The result reuses the memory in streams, but may be of different length and
// in any order.
//
// If a given stream has both an ID and a login, the ID is used and the login
// is replaced with the result from the API.
// If a stream has neither, it is ignored.
func UserStreams(ctx context.Context, client Client, tok *oauth2.Token, streams []Stream) ([]Stream, error) {
	v := url.Values{
		"user_id":    nil,
		"user_login": nil,
	}
	for _, s := range streams {
		if s.UserID != "" {
			v["user_id"] = append(v["user_id"], s.UserID)
			continue
		}
		if s.UserLogin != "" {
			v["user_login"] = append(v["user_login"], s.UserLogin)
		}
	}
	url := apiurl("/helix/streams", v)
	err := reqjson(ctx, client, tok, "GET", url, nil, &streams)
	if err != nil {
		return nil, fmt.Errorf("couldn't get streams info: %w", err)
	}
	return streams, nil
}
