package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *DuckDbStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenDuckDb(path)
	if err != nil {
		t.Fatalf("OpenDuckDb() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSaveScore_Dedup(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	rec := ScoreRecord{ShareLink: "https://putt.day/s/abc", Hole: 65, Strokes: 6, UserID: "u1", RecordedAt: time.Now()}

	inserted, err := s.SaveScore(ctx, rec)
	if err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}
	if !inserted {
		t.Fatalf("SaveScore() inserted = false, want true on first insert")
	}

	inserted, err = s.SaveScore(ctx, rec)
	if err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}
	if inserted {
		t.Fatalf("SaveScore() inserted = true, want false on duplicate")
	}

	scores, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(scores) != 1 {
		t.Fatalf("TopScores() len = %d, want 1", len(scores))
	}
}

func TestTopScores_Ordering(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	strokes := []int{5, 3, 9, 3}
	base := time.Now()
	for i, st := range strokes {
		rec := ScoreRecord{
			ShareLink:  filepath.Join("link", string(rune('a'+i))),
			Hole:       1,
			Strokes:    st,
			UserID:     string(rune('a' + i)),
			RecordedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if _, err := s.SaveScore(ctx, rec); err != nil {
			t.Fatalf("SaveScore() error = %v", err)
		}
	}

	scores, err := s.TopScores(ctx, 1, 3)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(scores) != 3 {
		t.Fatalf("TopScores() len = %d, want 3", len(scores))
	}
	want := []int{3, 3, 5}
	for i, rec := range scores {
		if rec.Strokes != want[i] {
			t.Errorf("TopScores()[%d].Strokes = %d, want %d", i, rec.Strokes, want[i])
		}
	}

	scores, err = s.TopScores(ctx, 1, 100)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(scores) != 4 {
		t.Fatalf("TopScores() with large limit len = %d, want 4", len(scores))
	}
}

func TestTopScores_UnknownHole(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	scores, err := s.TopScores(ctx, 999, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if scores != nil {
		t.Fatalf("TopScores() on unknown hole = %v, want nil", scores)
	}
}

func TestUserScores(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	entries := []struct {
		link    string
		userID  string
		strokes int
	}{
		{"https://putt.day/s/1", "u1", 7},
		{"https://putt.day/s/2", "u2", 4},
		{"https://putt.day/s/3", "u1", 5},
	}
	for i, e := range entries {
		rec := ScoreRecord{
			ShareLink:  e.link,
			Hole:       42,
			Strokes:    e.strokes,
			UserID:     e.userID,
			RecordedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if _, err := s.SaveScore(ctx, rec); err != nil {
			t.Fatalf("SaveScore() error = %v", err)
		}
	}

	scores, err := s.UserScores(ctx, 42, "u1", 10)
	if err != nil {
		t.Fatalf("UserScores() error = %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("UserScores() len = %d, want 2", len(scores))
	}
	if scores[0].Strokes != 5 || scores[1].Strokes != 7 {
		t.Errorf("UserScores() = %+v, want ascending [5, 7]", scores)
	}
}

func TestSaveScore_RepeatScoreCollapsesToMostRecent(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	first := ScoreRecord{ShareLink: "https://putt.day/s/first", Hole: 65, Strokes: 2, UserID: "u1", RecordedAt: base}
	second := ScoreRecord{ShareLink: "https://putt.day/s/second", Hole: 65, Strokes: 2, UserID: "u1", RecordedAt: base.Add(time.Hour)}

	if _, err := s.SaveScore(ctx, first); err != nil {
		t.Fatalf("SaveScore(first) error = %v", err)
	}
	inserted, err := s.SaveScore(ctx, second)
	if err != nil {
		t.Fatalf("SaveScore(second) error = %v", err)
	}
	if !inserted {
		t.Fatalf("SaveScore(second) inserted = false, want true (distinct share link, even though same score)")
	}

	top, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("TopScores() len = %d, want 1 (repeat score for the same user should collapse)", len(top))
	}
	if top[0].ShareLink != second.ShareLink {
		t.Errorf("TopScores()[0].ShareLink = %q, want %q (most recent play)", top[0].ShareLink, second.ShareLink)
	}

	// Unlike TopScores, UserScores is personal history: it does not collapse
	// repeat plays at the same score, so both links remain visible.
	userScores, err := s.UserScores(ctx, 65, "u1", 10)
	if err != nil {
		t.Fatalf("UserScores() error = %v", err)
	}
	if len(userScores) != 2 {
		t.Fatalf("UserScores() len = %d, want 2 (personal history keeps every play)", len(userScores))
	}
	gotLinks := map[string]bool{userScores[0].ShareLink: true, userScores[1].ShareLink: true}
	if !gotLinks[first.ShareLink] || !gotLinks[second.ShareLink] {
		t.Errorf("UserScores() = %+v, want both %q and %q", userScores, first.ShareLink, second.ShareLink)
	}

	// The superseded share link is still known to Exists (it really was
	// submitted), it just no longer surfaces in leaderboard queries.
	exists, err := s.Exists(ctx, first.ShareLink)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Errorf("Exists(first) = false, want true")
	}
}

func TestSaveScore_DifferentScoresForSameUserBothAppear(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	entries := []ScoreRecord{
		{ShareLink: "https://putt.day/s/a", Hole: 65, Strokes: 2, UserID: "u1", RecordedAt: base},
		{ShareLink: "https://putt.day/s/b", Hole: 65, Strokes: 4, UserID: "u1", RecordedAt: base.Add(time.Minute)},
	}
	for _, rec := range entries {
		if _, err := s.SaveScore(ctx, rec); err != nil {
			t.Fatalf("SaveScore() error = %v", err)
		}
	}

	scores, err := s.UserScores(ctx, 65, "u1", 10)
	if err != nil {
		t.Fatalf("UserScores() error = %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("UserScores() len = %d, want 2 (distinct scores should not collapse)", len(scores))
	}
}

func TestDeleteByShareLink(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	rec := ScoreRecord{ShareLink: "https://putt.day/s/abc", Hole: 65, Strokes: 6, UserID: "u1", RecordedAt: time.Now()}
	if _, err := s.SaveScore(ctx, rec); err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}

	removed, err := s.DeleteByShareLink(ctx, rec.ShareLink, "u1", false)
	if err != nil {
		t.Fatalf("DeleteByShareLink() error = %v", err)
	}
	if !removed {
		t.Fatalf("DeleteByShareLink() removed = false, want true")
	}

	top, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(top) != 0 {
		t.Fatalf("TopScores() len = %d, want 0 after delete", len(top))
	}
	userScores, err := s.UserScores(ctx, 65, "u1", 10)
	if err != nil {
		t.Fatalf("UserScores() error = %v", err)
	}
	if len(userScores) != 0 {
		t.Fatalf("UserScores() len = %d, want 0 after delete", len(userScores))
	}

	exists, err := s.Exists(ctx, rec.ShareLink)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Fatalf("Exists() = true, want false after delete")
	}
}

func TestDeleteByShareLink_NotFound(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	_, err := s.DeleteByShareLink(ctx, "https://putt.day/s/nope", "u1", false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteByShareLink() error = %v, want ErrNotFound", err)
	}
}

func TestDeleteByShareLink_Forbidden(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	rec := ScoreRecord{ShareLink: "https://putt.day/s/abc", Hole: 65, Strokes: 6, UserID: "u1", RecordedAt: time.Now()}
	if _, err := s.SaveScore(ctx, rec); err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}

	_, err := s.DeleteByShareLink(ctx, rec.ShareLink, "someone-else", false)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("DeleteByShareLink() error = %v, want ErrForbidden", err)
	}

	// The record must still be there — a forbidden attempt shouldn't delete anything.
	top, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(top) != 1 {
		t.Fatalf("TopScores() len = %d, want 1 (unauthorized delete must not remove the record)", len(top))
	}
}

func TestDeleteByShareLink_AdminBypassesOwnership(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	rec := ScoreRecord{ShareLink: "https://putt.day/s/abc", Hole: 65, Strokes: 6, UserID: "u1", RecordedAt: time.Now()}
	if _, err := s.SaveScore(ctx, rec); err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}

	removed, err := s.DeleteByShareLink(ctx, rec.ShareLink, "an-admin-who-does-not-own-this", true)
	if err != nil {
		t.Fatalf("DeleteByShareLink() error = %v", err)
	}
	if !removed {
		t.Fatalf("DeleteByShareLink() removed = false, want true when bypassOwnership is set")
	}

	top, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(top) != 0 {
		t.Fatalf("TopScores() len = %d, want 0 after admin delete", len(top))
	}
}

func TestDeleteByShareLink_SupersededLinkDoesNotClobberNewerRecord(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	first := ScoreRecord{ShareLink: "https://putt.day/s/first", Hole: 65, Strokes: 2, UserID: "u1", RecordedAt: base}
	second := ScoreRecord{ShareLink: "https://putt.day/s/second", Hole: 65, Strokes: 2, UserID: "u1", RecordedAt: base.Add(time.Hour)}

	if _, err := s.SaveScore(ctx, first); err != nil {
		t.Fatalf("SaveScore(first) error = %v", err)
	}
	if _, err := s.SaveScore(ctx, second); err != nil {
		t.Fatalf("SaveScore(second) error = %v", err)
	}

	// "first" no longer appears in leaderboard queries (collapsed by "second"),
	// but its link is still individually removable/known to Exists.
	removed, err := s.DeleteByShareLink(ctx, first.ShareLink, "u1", false)
	if err != nil {
		t.Fatalf("DeleteByShareLink(first) error = %v", err)
	}
	if removed {
		t.Fatalf("DeleteByShareLink(first) removed = true, want false: it was already superseded and never affected the leaderboard")
	}

	// "second" must be completely unaffected by deleting the superseded "first".
	top, err := s.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(top) != 1 || top[0].ShareLink != second.ShareLink {
		t.Fatalf("TopScores() = %+v, want a single entry for %q", top, second.ShareLink)
	}

	firstExists, err := s.Exists(ctx, first.ShareLink)
	if err != nil {
		t.Fatalf("Exists(first) error = %v", err)
	}
	if firstExists {
		t.Errorf("Exists(first) = true, want false after deleting it")
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	exists, err := s.Exists(ctx, "https://putt.day/s/none")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Fatalf("Exists() = true, want false before save")
	}

	rec := ScoreRecord{ShareLink: "https://putt.day/s/none", Hole: 1, Strokes: 3, UserID: "u1", RecordedAt: time.Now()}
	if _, err := s.SaveScore(ctx, rec); err != nil {
		t.Fatalf("SaveScore() error = %v", err)
	}

	exists, err = s.Exists(ctx, "https://putt.day/s/none")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Fatalf("Exists() = false, want true after save")
	}
}

func TestListHoles_Ordering(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	holes := []int{3, 1, 2}
	for i, h := range holes {
		rec := ScoreRecord{
			ShareLink:  filepath.Join("link", string(rune('a'+i))),
			Hole:       h,
			Strokes:    4,
			UserID:     "u1",
			RecordedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if _, err := s.SaveScore(ctx, rec); err != nil {
			t.Fatalf("SaveScore() error = %v", err)
		}
	}

	got, err := s.ListHoles(ctx, 10)
	if err != nil {
		t.Fatalf("ListHoles() error = %v", err)
	}
	want := []int{3, 2, 1}
	if len(got) != len(want) {
		t.Fatalf("ListHoles() len = %d, want %d", len(got), len(want))
	}
	for i, h := range got {
		if h.Number != want[i] {
			t.Errorf("ListHoles()[%d].Number = %d, want %d", i, h.Number, want[i])
		}
	}

	got, err = s.ListHoles(ctx, 2)
	if err != nil {
		t.Fatalf("ListHoles() with limit error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListHoles() with limit len = %d, want 2", len(got))
	}
	if got[0].Number != 3 || got[1].Number != 2 {
		t.Fatalf("ListHoles() with limit = %v, want [3, 2]", got)
	}
}

func TestListHoles_TopStrokes(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	base := time.Now()
	strokes := []int{7, 3, 5}
	for i, st := range strokes {
		rec := ScoreRecord{
			ShareLink:  filepath.Join("link", string(rune('a'+i))),
			Hole:       10,
			Strokes:    st,
			UserID:     string(rune('a' + i)),
			RecordedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if _, err := s.SaveScore(ctx, rec); err != nil {
			t.Fatalf("SaveScore() error = %v", err)
		}
	}
	got, err := s.ListHoles(ctx, 10)
	if err != nil {
		t.Fatalf("ListHoles() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListHoles() len = %d, want 1", len(got))
	}
	if got[0].TopStrokes == nil || *got[0].TopStrokes != 3 {
		t.Fatalf("ListHoles()[0].TopStrokes = %v, want 3", got[0].TopStrokes)
	}
}

func TestListHoles_TopStrokesNilWithoutScores(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)

	if _, err := s.db.ExecContext(ctx, `INSERT INTO holes (hole) VALUES (99)`); err != nil {
		t.Fatalf("pre-registering hole: %v", err)
	}

	got, err := s.ListHoles(ctx, 10)
	if err != nil {
		t.Fatalf("ListHoles() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListHoles() len = %d, want 1", len(got))
	}
	if got[0].TopStrokes != nil {
		t.Fatalf("ListHoles()[0].TopStrokes = %v, want nil", *got[0].TopStrokes)
	}
}
