package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/duckdb/duckdb-go/v2"
)

type DuckDbStore struct {
	db *sql.DB
}

// compile time interface implementation check
var _ Store = (*DuckDbStore)(nil)

func OpenDuckDb(path string) (*DuckDbStore, error) {
	connector, err := duckdb.NewConnector(path, nil)
	if err != nil {
		return nil, fmt.Errorf("opening duckdb: %w", err)
	}
	ddbs := &DuckDbStore{
		db: sql.OpenDB(connector),
	}

	if err := initializeDuckDbStore(ddbs); err != nil {
		_ = ddbs.Close()
		return nil, fmt.Errorf("initializing duckdb schema: %w", err)
	}
	return ddbs, nil
}

// schema defines the two tables backing DuckDbStore:
//
//   - holes is a dimension table for hole metadata (currently just the
//     custom flag) that needs to exist independently of any recorded score,
//     e.g. an admin pre-registering a custom hole before anyone plays it.
//   - scores is the fact table, one row per recorded ScoreRecord, holding
//     user/guild/channel as plain reference IDs since Discord — not this
//     store — owns their canonical data.
const schema = `
CREATE TABLE IF NOT EXISTS holes (
    hole   INTEGER PRIMARY KEY,
    custom BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS scores (
    share_link  TEXT PRIMARY KEY,
    hole        INTEGER NOT NULL REFERENCES holes(hole),
    strokes     INTEGER NOT NULL,
    user_id     TEXT NOT NULL,
    guild_id    TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    message_id  TEXT NOT NULL,
    recorded_at TIMESTAMP NOT NULL
);
`

func initializeDuckDbStore(store *DuckDbStore) error {
	_, err := store.db.Exec(schema)
	return err
}

// Close implements [Store].
func (d *DuckDbStore) Close() error {
	return d.db.Close()
}

// DeleteByShareLink implements [Store].
func (d *DuckDbStore) DeleteByShareLink(ctx context.Context, shareLink string, requestingUserID string, bypassOwnership bool) (removedFromLeaderboard bool, err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	var hole, strokes int
	var userID string
	var recordedAt time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT hole, strokes, user_id, recorded_at FROM scores WHERE share_link = ?`,
		shareLink,
	).Scan(&hole, &strokes, &userID, &recordedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("looking up share link %q: %w", shareLink, err)
	}

	if !bypassOwnership && userID != requestingUserID {
		return false, ErrForbidden
	}

	// isCurrent reports whether shareLink is the most recent play for its
	// (hole, user, strokes) group — i.e. the one TopScores/UserScores'
	// dedup would surface — as opposed to one already superseded by a
	// newer play with the same score.
	var isCurrent bool
	err = tx.QueryRowContext(ctx,
		`SELECT NOT EXISTS (
			SELECT 1 FROM scores
			WHERE hole = ? AND user_id = ? AND strokes = ? AND recorded_at > ?
		)`,
		hole, userID, strokes, recordedAt,
	).Scan(&isCurrent)
	if err != nil {
		return false, fmt.Errorf("checking leaderboard currency: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM scores WHERE share_link = ?`, shareLink); err != nil {
		return false, fmt.Errorf("deleting share link %q: %w", shareLink, err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing delete: %w", err)
	}
	return isCurrent, nil
}

// Exists implements [Store].
func (d *DuckDbStore) Exists(ctx context.Context, shareLink string) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM scores WHERE share_link = ?)`,
		shareLink,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking share link %q existence: %w", shareLink, err)
	}
	return exists, nil
}

// SaveScore implements [Store].
func (d *DuckDbStore) SaveScore(ctx context.Context, rec ScoreRecord) (inserted bool, err error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO holes (hole) VALUES (?) ON CONFLICT DO NOTHING`,
		rec.Hole,
	); err != nil {
		return false, fmt.Errorf("registering hole: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`
	INSERT INTO scores (share_link, hole, strokes, user_id, guild_id, channel_id, message_id, recorded_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (share_link) DO NOTHING
	`, rec.ShareLink, rec.Hole, rec.Strokes, rec.UserID, rec.GuildID, rec.ChannelID, rec.MessageID, rec.RecordedAt,
	)
	if err != nil {
		return false, fmt.Errorf("saving score: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking affected rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing score: %w", err)
	}
	return affected > 0, nil
}

// TopScores implements [Store].
func (d *DuckDbStore) TopScores(ctx context.Context, hole int, limit int) ([]ScoreRecord, error) {
	rows, err := d.db.QueryContext(ctx, `
	SELECT share_link, hole, strokes, user_id, guild_id, channel_id, message_id, recorded_at
	FROM (
		SELECT *,
		       ROW_NUMBER() OVER (PARTITION BY hole, user_id, strokes ORDER BY recorded_at DESC) AS rn
		FROM scores
		WHERE hole = ?
	) ranked
	WHERE rn = 1
	ORDER BY strokes ASC
	LIMIT ?
	`, hole, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top scores: %w", err)
	}
	defer rows.Close()

	var out []ScoreRecord
	for rows.Next() {
		var record ScoreRecord
		if err := rows.Scan(&record.ShareLink, &record.Hole, &record.Strokes, &record.UserID, &record.GuildID, &record.ChannelID, &record.MessageID, &record.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning top score: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating top scores: %w", err)
	}
	return out, nil
}

// UserScores implements [Store].
func (d *DuckDbStore) UserScores(ctx context.Context, hole int, userID string, limit int) ([]ScoreRecord, error) {
	rows, err := d.db.QueryContext(ctx, `
	SELECT share_link, hole, strokes, user_id, guild_id, channel_id, message_id, recorded_at
	FROM scores
	WHERE hole = ? AND user_id = ? 
	ORDER BY strokes ASC
	LIMIT ? 
	`, hole, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying user scores: %w", err)
	}
	defer rows.Close()

	var out []ScoreRecord
	for rows.Next() {
		var record ScoreRecord
		if err := rows.Scan(&record.ShareLink, &record.Hole, &record.Strokes, &record.UserID, &record.GuildID, &record.ChannelID, &record.MessageID, &record.RecordedAt); err != nil {
			return nil, fmt.Errorf("scanning user score: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating user scores: %w", err)
	}
	return out, nil
}
