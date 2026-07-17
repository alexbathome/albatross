// Package api defines the wire-format request/response types for the
// albatross HTTP REST API, kept separate from [store.ScoreRecord] so the
// JSON contract can evolve independently of the storage layer.
package api

import "time"

// ScoreRecord is a single recorded putt.day play, as returned by the API.
type ScoreRecord struct {
	ShareLink  string    `json:"share_link"`
	Hole       int       `json:"hole"`
	Strokes    int       `json:"strokes"`
	UserID     string    `json:"user_id"`
	GuildID    string    `json:"guild_id"`
	ChannelID  string    `json:"channel_id"`
	MessageID  string    `json:"message_id"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ExistsResponse is the response body for GET /api/scores/exists.
type ExistsResponse struct {
	Exists bool `json:"exists"`
}

// ErrorResponse is the response body for any non-2xx response.
type ErrorResponse struct {
	Error string `json:"error"`
}
