package eventsub

import "github.com/go-json-experiment/json/jsontext"

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
	Transport Transport `json:"transport"`
	// Created is the time at which the subscription was created in
	// RFC3339Nano format.
	Created string `json:"created_at"`
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

type Transport struct {
	Session string `json:"session_id"`
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

type session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Keepalive int    `json:"keepalive_timeout_seconds"`
	Reconnect string `json:"reconnect_url"`
	Connected string `json:"connected_at"`
}
