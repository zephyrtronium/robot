/*
Copyright (C) 2020  Branden J Brown

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func fetchClientID(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", "OAuth "+token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error querying %v: %w", req.URL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	type expected struct {
		ClientID string `json:"client_id"`
		// UserID    string   `json:"user_id"`
		// Login     string   `json:"login"`
		// Scopes    []string `json:"scopes"`
		// ExpiresIn string   `json:"expires_in"`
	}
	var r expected
	if err = json.Unmarshal(body, &r); err != nil {
		err = fmt.Errorf("error decoding response: %w", err)
	}
	return r.ClientID, err
}

// online fetches the online status of a list of Twitch usernames. The returned
// keys are the user_name fields of the respective results, each converted to
// lowercase and prepended with a # character as should be the channel names to
// which the bot is connected.
//
// Per https://dev.twitch.tv/docs/api/reference#get-streams, this function
// cannot query more than 100 users.
func online(ctx context.Context, token, clientID string, channels []string) (map[string]bool, error) {
	if len(channels) == 0 {
		return nil, nil
	}

	v := url.Values{}
	v.Set("first", strconv.Itoa(len(channels)))
	for _, channel := range channels {
		v.Add("user_login", strings.TrimPrefix(channel, "#"))
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.twitch.tv/helix/streams", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.URL.RawQuery = v.Encode()
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Client-Id", clientID)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error querying %v: %w", req.URL, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	type expected struct {
		Data []struct {
			UserName string `json:"user_name"`
		} `json:"data"`
	}
	var r expected
	if err = json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	m := make(map[string]bool, len(r.Data))
	for _, v := range r.Data {
		m["#"+strings.ToLower(v.UserName)] = true
	}
	return m, nil
}
