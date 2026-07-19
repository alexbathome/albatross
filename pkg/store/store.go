// Package store persists and queries recorded putt.day scores.
package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a share link has no recorded score.
	ErrNotFound = errors.New("store: record not found")
	// ErrForbidden is returned when a share link's score is not owned by
	// the requesting user.
	ErrForbidden = errors.New("store: user does not own this record")
)

// ScoreRecord is a single recorded putt.day play.
type ScoreRecord struct {
	ShareLink string
	Hole      int
	Strokes   int
	UserID    string
	// Username is the Discord display name (global name if set, else
	// username) of UserID at the time the score was recorded. It is a
	// point-in-time snapshot, not kept in sync with later name changes.
	Username   string
	GuildID    string
	ChannelID  string
	MessageID  string
	RecordedAt time.Time
}

// Hole is a registered hole, present once someone has played it or an admin
// has pre-registered it.
type Hole struct {
	Number int
	Custom bool
	// TopStrokes is the lowest recorded stroke count for this hole, or nil
	// if it has no recorded scores (e.g. pre-registered but never played,
	// or every score for it has since been removed).
	TopStrokes *int
}

// Store persists and queries ScoreRecords.
type Store interface {
	// SaveScore persists rec. If rec.ShareLink has already been recorded,
	// it is a no-op: inserted is false and err is nil.
	SaveScore(ctx context.Context, rec ScoreRecord) (inserted bool, err error)

	// Exists reports whether shareLink has already been recorded.
	Exists(ctx context.Context, shareLink string) (bool, error)

	// DeleteByShareLink removes the record for shareLink. Returns ErrNotFound
	// if no record exists for shareLink. Unless bypassOwnership is true, it
	// also returns ErrForbidden if requestingUserID does not match the
	// record's owner — bypassOwnership lets a server admin remove a record
	// they don't own.
	//
	// removedFromLeaderboard reports whether the deletion actually changed
	// TopScores/UserScores output: if shareLink had already been superseded
	// by a more recent play with the same (hole, user, strokes) — see
	// SaveScore — the leaderboard entry belongs to that newer link and is
	// left untouched, even though shareLink's own record is still removed.
	DeleteByShareLink(ctx context.Context, shareLink string, requestingUserID string, bypassOwnership bool) (removedFromLeaderboard bool, err error)

	// TopScores returns up to limit records for hole, ascending by strokes.
	TopScores(ctx context.Context, hole int, limit int) ([]ScoreRecord, error)

	// UserScores returns up to limit of userID's records for hole, ascending
	// by strokes. Unlike TopScores, results are not deduplicated by
	// (hole, user, strokes): repeat plays at the same score are all
	// returned, so a user can browse back through every run on a hole —
	// including a fun one worth keeping — not just their best per score.
	UserScores(ctx context.Context, hole int, userID string, limit int) ([]ScoreRecord, error)

	// ListHoles returns up to limit registered holes, descending by hole
	// number (most recently added hole first).
	ListHoles(ctx context.Context, limit int) ([]Hole, error)

	// Close releases any resources held by the store.
	Close() error
}
