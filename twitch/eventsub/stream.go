package eventsub

// StreamOnline is the payload for a stream.online or stream.offline notification.
type Stream struct {
	// ID is the stream ID.
	// Only present in stream.online messages.
	ID string `json:"id"`
	// Broadcaster is the broadcaster's user ID.
	Broadcaster string `json:"broadcaster_user_id"`
	// BroadcasterLogin is the broadcaster's user login.
	BroadcasterLogin string `json:"broadcaster_user_login"`
	// BoradcasterName is the broadcaster's display name.
	BroadcasterName string `json:"broadcaster_user_name"`
	// Type is the stream type, usually "live".
	Type string `json:"type"`
	// Started is the time at which the stream started in RFC3339Nano format.
	Started string `json:"started_at"`
}
