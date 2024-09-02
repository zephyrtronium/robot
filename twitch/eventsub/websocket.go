// Package eventsub implements low-level EventSub WebSocket operations.
package eventsub

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

// Session is an EventSub WebSocket connection.
type Session struct {
	// conn is the actual connection.
	conn *websocket.Conn
	// id is the session ID.
	id string
	// timeout is the timeout between reads.
	timeout time.Duration
}

// message is a generic message received from EventSub.
type message struct {
	Metadata metadata `json:"metadata"`
	Payload  payload  `json:"payload"`
}

type payload struct {
	Subscription Subscription   `json:"subscription"`
	Session      session        `json:"session"`
	Event        jsontext.Value `json:"event"`
}

// metadata is WebSocket message metadata.
type metadata struct {
	// ID is the message UUID.
	ID string `json:"message_id"`
	// Type is the type of the associated payload.
	Type string `json:"message_type"`
	// Timestamp is the message time in RFC3339Nano format.
	Timestamp string `json:"message_timestamp"`
	// SubscriptionType is the subscription type for notification messages.
	SubscriptionType string `json:"subscription_type"`
	// SubscriptionVersion is the version of the subscription type.
	SubscriptionVersion string `json:"subscription_version"`
}

// Event is a payload of a notification message.
type Event struct {
	Subscription Subscription   `json:"subscription"`
	Event        jsontext.Value `json:"event"`
}

type Subscription struct {
	// ID is the subscription ID.
	ID string `json:"id"`
	// Status is the subscription status.
	// For event notifications, this is always "enabled".
	Status string `json:"status"`
	// Type is the event type.
	Type string `json:"type"`
	// Version is the version of the event type.
	Version string `json:"version"`
	// Cost is the event cost.
	Cost int `json:"cost"`
	// Condition is the event condition under which the event fired.
	Condition Condition `json:"condition"`
	// Transport holds the WebSocket connection details.
	Transport struct {
		Session string `json:"session_id"`
	} `json:"transport"`
	// Created is the time at which the subscription was created in
	// RFC3339Nano format.
	Created string `json:"created_at"`
}

type session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Keepalive int    `json:"keepalive_timeout_seconds"`
	Reconnect string `json:"reconnect_url"`
	Connected string `json:"connected_at"`
}

type Condition struct {
	// Broadcaster is the broadcaster user ID associated with the event origin.
	// Not all messages have a broadcaster ID.
	// Note that messages which use broadcaster_id instead of broadcaster_user_id
	// will be captured in Extra instead.
	Broadcaster string `json:"broadcaster_user_id"`
	// User is the user ID associated with receiving the event.
	// Not all messages have a user ID.
	User string `json:"user_id"`
	// Extra holds any additional fields in the condition.
	Extra jsontext.Value `json:",unknown"`
}

// Connect connects to the Twitch EventSub server.
// If the HTTP client is nil, [http.DefaultClient] is used instead.
// keepalive is the interval in seconds to request keepalive messages.
// If zero, the Twitch default is used.
// url may be a reconnect URL given by a previous EventSub connection.
// If empty, the default Twitch EventSub URL is used.
func Connect(ctx context.Context, client *http.Client, keepalive int, url string) (*Session, error) {
	var opts *websocket.DialOptions
	if client != nil {
		opts = &websocket.DialOptions{
			HTTPClient: client,
		}
	}
	if url == "" {
		url = "wss://eventsub.wss.twitch.tv/ws"
	}
	if keepalive != 0 {
		url += "?keepalive_timeout_seconds=" + strconv.Itoa(keepalive)
	}

	slog.DebugContext(ctx, "dial EventSub", slog.String("url", url))
	conn, resp, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		if resp != nil {
			b := make([]byte, 1024)
			n, _ := resp.Body.Read(b)
			b = b[:n]
			return nil, fmt.Errorf("couldn't connect to EventSub: %w (%s)", err, b)
		}
		return nil, fmt.Errorf("couldn't connect to EventSub: %w", err)
	}

	// The first message is a welcome.
	_, m, err := conn.Read(ctx)
	if err != nil {
		conn.CloseNow()
		return nil, fmt.Errorf("couldn't receive welcome: %w", err)
	}
	var msg message
	if err := json.Unmarshal(m, &msg); err != nil {
		conn.CloseNow()
		return nil, fmt.Errorf("couldn't decode welcome: %w", err)
	}
	if msg.Metadata.Type != "session_welcome" {
		conn.CloseNow()
		return nil, fmt.Errorf("invalid welcome message with type %q", msg.Metadata.Type)
	}
	s := &Session{
		conn:    conn,
		id:      msg.Payload.Session.ID,
		timeout: time.Duration(msg.Payload.Session.Keepalive+2) * time.Second, // add 2 seconds
	}
	return s, nil
}

// ID returns the EventSub session ID.
func (s *Session) ID() string {
	return s.id
}

// Recv gets the next notification message.
// Keepalive messages are handled transparently.
// The error may be of type [ReconnectError], giving a reconnect URL.
//
// Note that the context becoming done during a call to Recv will cause the
// WebSocket connection to close as well.
func (s *Session) Recv(ctx context.Context) (*Event, error) {
	for {
		tctx, cancel := context.WithTimeout(ctx, s.timeout)
		_, m, err := s.conn.Read(tctx)
		cancel()
		if err != nil {
			return nil, err
		}
		var msg message
		if err := json.Unmarshal(m, &msg); err != nil {
			return nil, fmt.Errorf("couldn't decode message %q: %w", m, err)
		}
		switch msg.Metadata.Type {
		case "notification":
			slog.DebugContext(ctx, "EventSub notification",
				slog.String("id", msg.Metadata.ID),
				slog.String("subscription_type", msg.Metadata.SubscriptionType),
				slog.String("subscription_version", msg.Metadata.SubscriptionVersion),
			)
			return &Event{Subscription: msg.Payload.Subscription, Event: msg.Payload.Event}, nil

		case "session_keepalive":
			slog.DebugContext(ctx, "EventSub keepalive", slog.String("id", msg.Metadata.ID))
			continue

		case "session_reconnect":
			slog.DebugContext(ctx, "EventSub reconnect", slog.String("id", msg.Metadata.ID))
			r := &ReconnectError{
				Session:      msg.Payload.Session.ID,
				ReconnectURL: msg.Payload.Session.Reconnect,
				Connected:    msg.Payload.Session.Connected,
			}
			return nil, r

		case "revocation":
			slog.DebugContext(ctx, "EventSub revocation",
				slog.String("id", msg.Metadata.ID),
				slog.String("subscription_type", msg.Metadata.SubscriptionType),
				slog.String("subscription_version", msg.Metadata.SubscriptionVersion),
			)
			r := &RevocationError{
				Subscription: msg.Payload.Subscription.ID,
				Status:       msg.Payload.Subscription.Status,
				Type:         msg.Payload.Subscription.Type,
				Version:      msg.Payload.Subscription.Version,
				Created:      msg.Payload.Subscription.Created,
			}
			return nil, r
		}
	}
}

// Close ends the WebSocket session.
func (s *Session) Close() error {
	return s.conn.CloseNow()
}

// ReconnectError is an error representing a WebSocket reconnect message.
type ReconnectError struct {
	// Session is the session ID of the reconnecting session.
	Session string `json:"id"`
	// ReconnectURL is the URL sent by EventSub to reconnect.
	ReconnectURL string `json:"reconnect_url"`
	// Connected is the time at which the connection was originally created
	// as an RFC3339Nano string.
	Connected string `json:"connected_at"`
}

func (err *ReconnectError) Error() string {
	return fmt.Sprintf("reconnect session %s created %s", err.Session, err.Connected)
}

// RevocationError is an error representing a WebSocket revocation message.
type RevocationError struct {
	// Subscription is the subscription ID of the revoked subscription.
	Subscription string `json:"id"`
	// Status is the reason for the revocation.
	Status string `json:"status"`
	// Type is the subscription type.
	Type string `json:"type"`
	// Version is the version of the subscription type.
	Version string `json:"version"`
	// Created is the time at which the subscription was originally created
	// as an RFC3339Nano string.
	Created string `json:"created_at"`
}

func (err *RevocationError) Error() string {
	return fmt.Sprintf("%s/%s subscription %s revoked: %s", err.Type, err.Version, err.Subscription, err.Status)
}
