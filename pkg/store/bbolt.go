package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketLinks = []byte("links")
	bucketHoles = []byte("holes")
	bucketUsers = []byte("users")
)

// BoltStore is a Store backed by a bbolt database file.
type BoltStore struct {
	db *bolt.DB
}

// Open opens (creating if necessary) a bbolt database at path and prepares
// the buckets BoltStore relies on.
func Open(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening bbolt db: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{bucketLinks, bucketHoles, bucketUsers} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("creating bucket %s: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &BoltStore{db: db}, nil
}

func holeKey(hole int) []byte {
	return fmt.Appendf(nil, "%04d", hole)
}

// scoreKey identifies a (strokes, userID) pair within a hole bucket. Reusing
// the same key for a repeat score is intentional: a later play with the same
// strokes overwrites the earlier one, so each user contributes at most one
// row per distinct score on a hole.
func scoreKey(strokes int, userID string) []byte {
	return fmt.Appendf(nil, "%04d-%s", strokes, userID)
}

// userScoreKey identifies a strokes value within a user's per-hole bucket.
// Like scoreKey, reusing the key on a repeat score is intentional overwrite.
func userScoreKey(strokes int) []byte {
	return fmt.Appendf(nil, "%04d", strokes)
}

// SaveScore implements Store, writing rec into the links, holes, and users
// buckets that back the various lookups.
func (s *BoltStore) SaveScore(ctx context.Context, rec ScoreRecord) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return false, fmt.Errorf("marshaling score record: %w", err)
	}

	inserted := false
	err = s.db.Update(func(tx *bolt.Tx) error {
		links := tx.Bucket(bucketLinks)
		if links.Get([]byte(rec.ShareLink)) != nil {
			return nil
		}
		if err := links.Put([]byte(rec.ShareLink), data); err != nil {
			return err
		}

		holeBucket, err := tx.Bucket(bucketHoles).CreateBucketIfNotExists(holeKey(rec.Hole))
		if err != nil {
			return err
		}
		if err := holeBucket.Put(scoreKey(rec.Strokes, rec.UserID), data); err != nil {
			return err
		}

		userBucket, err := tx.Bucket(bucketUsers).CreateBucketIfNotExists([]byte(rec.UserID))
		if err != nil {
			return err
		}
		userHoleBucket, err := userBucket.CreateBucketIfNotExists(holeKey(rec.Hole))
		if err != nil {
			return err
		}
		if err := userHoleBucket.Put(userScoreKey(rec.Strokes), data); err != nil {
			return err
		}

		inserted = true
		return nil
	})
	return inserted, err
}

// DeleteByShareLink implements Store, removing shareLink's record from every
// bucket it was written to.
func (s *BoltStore) DeleteByShareLink(ctx context.Context, shareLink string, requestingUserID string, bypassOwnership bool) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	removedFromLeaderboard := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		links := tx.Bucket(bucketLinks)
		data := links.Get([]byte(shareLink))
		if data == nil {
			return ErrNotFound
		}
		var rec ScoreRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			return fmt.Errorf("unmarshaling score record: %w", err)
		}
		if !bypassOwnership && rec.UserID != requestingUserID {
			return ErrForbidden
		}
		if err := links.Delete([]byte(shareLink)); err != nil {
			return err
		}

		holeBucket := tx.Bucket(bucketHoles).Bucket(holeKey(rec.Hole))
		if holeBucket == nil {
			return nil
		}
		key := scoreKey(rec.Strokes, rec.UserID)
		current := holeBucket.Get(key)
		if current == nil {
			return nil
		}
		var currentRec ScoreRecord
		if err := json.Unmarshal(current, &currentRec); err != nil {
			return fmt.Errorf("unmarshaling score record: %w", err)
		}
		if currentRec.ShareLink != shareLink {
			// shareLink was already superseded by a newer play with the same
			// (hole, user, strokes); the leaderboard entry belongs to that
			// newer link, so leave it in place.
			return nil
		}
		if err := holeBucket.Delete(key); err != nil {
			return err
		}
		if userBucket := tx.Bucket(bucketUsers).Bucket([]byte(rec.UserID)); userBucket != nil {
			if userHoleBucket := userBucket.Bucket(holeKey(rec.Hole)); userHoleBucket != nil {
				if err := userHoleBucket.Delete(userScoreKey(rec.Strokes)); err != nil {
					return err
				}
			}
		}
		removedFromLeaderboard = true
		return nil
	})
	return removedFromLeaderboard, err
}

// Exists implements Store.
func (s *BoltStore) Exists(ctx context.Context, shareLink string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		exists = tx.Bucket(bucketLinks).Get([]byte(shareLink)) != nil
		return nil
	})
	return exists, err
}

// TopScores implements Store, reading the hole bucket in key (ascending
// strokes) order.
func (s *BoltStore) TopScores(ctx context.Context, hole int, limit int) ([]ScoreRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []ScoreRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketHoles).Bucket(holeKey(hole))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil && len(out) < limit; k, v = c.Next() {
			var rec ScoreRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				return fmt.Errorf("unmarshaling score record: %w", err)
			}
			out = append(out, rec)
		}
		return nil
	})
	return out, err
}

// UserScores implements Store, reading userID's per-hole bucket in key
// (ascending strokes) order.
func (s *BoltStore) UserScores(ctx context.Context, hole int, userID string) ([]ScoreRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []ScoreRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket(bucketUsers).Bucket([]byte(userID))
		if userBucket == nil {
			return nil
		}
		holeBucket := userBucket.Bucket(holeKey(hole))
		if holeBucket == nil {
			return nil
		}
		c := holeBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var rec ScoreRecord
			if err := json.Unmarshal(v, &rec); err != nil {
				return fmt.Errorf("unmarshaling score record: %w", err)
			}
			out = append(out, rec)
		}
		return nil
	})
	return out, err
}

// Close implements Store by closing the underlying bbolt database.
func (s *BoltStore) Close() error {
	return s.db.Close()
}
