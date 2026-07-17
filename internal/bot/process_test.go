package bot

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/alexbathome/albatross/pkg/puttday"
	"github.com/alexbathome/albatross/pkg/store"
)

type fakeCollector struct {
	score *puttday.SharedScore
	err   error
}

func (f *fakeCollector) Collect(ctx context.Context, shareLink string) (*puttday.SharedScore, error) {
	return f.score, f.err
}

func openTestStore(t *testing.T) *store.DuckDbStore {
	t.Helper()
	s, err := store.OpenDuckDb(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.OpenDuckDb() error = %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestProcessMessage_NoLink(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{}

	msg := IncomingMessage{Content: "just chatting", AuthorID: "u1"}
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != "" {
		t.Errorf("ProcessMessage() = %q, want empty (no reaction)", got)
	}
}

func TestProcessMessage_BotAuthorIgnored(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{score: &puttday.SharedScore{Hole: 65, Strokes: 6, Link: "https://putt.day/s/abc"}}

	msg := IncomingMessage{Content: "https://putt.day/s/abc", AuthorID: "bot1", IsBot: true}
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != "" {
		t.Errorf("ProcessMessage() = %q, want empty for bot author", got)
	}
	scores, _ := st.TopScores(ctx, 65, 10)
	if len(scores) != 0 {
		t.Errorf("TopScores() = %v, want empty since bot messages aren't recorded", scores)
	}
}

func TestProcessMessage_GuildFilter(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{score: &puttday.SharedScore{Hole: 65, Strokes: 6, Link: "https://putt.day/s/abc"}}

	tests := []struct {
		name           string
		msgGuildID     string
		allowedGuildID string
		want           string
	}{
		{"matching guild is processed", "g1", "g1", reactionSuccess},
		{"other guild is dropped", "g2", "g1", ""},
		{"DM is dropped when a guild is configured", "", "g1", ""},
		{"empty allowed guild processes any guild", "g2", "", reactionSuccess},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := IncomingMessage{Content: "https://putt.day/s/abc", AuthorID: "u1", GuildID: tt.msgGuildID}
			got := ProcessMessage(ctx, collector, st, msg, tt.allowedGuildID)
			if got != tt.want {
				t.Errorf("ProcessMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessMessage_Success(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{score: &puttday.SharedScore{Hole: 65, Strokes: 6, Link: "https://putt.day/s/abc"}}

	msg := IncomingMessage{Content: "https://putt.day/s/abc", AuthorID: "u1"}
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != reactionSuccess {
		t.Errorf("ProcessMessage() = %q, want %q", got, reactionSuccess)
	}

	scores, err := st.TopScores(ctx, 65, 10)
	if err != nil {
		t.Fatalf("TopScores() error = %v", err)
	}
	if len(scores) != 1 || scores[0].UserID != "u1" || scores[0].Strokes != 6 {
		t.Errorf("TopScores() = %+v, want single record for u1 with 6 strokes", scores)
	}
}

func TestProcessMessage_CollectError(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{err: errors.New("boom")}

	msg := IncomingMessage{Content: "https://putt.day/s/abc", AuthorID: "u1"}
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != reactionFailure {
		t.Errorf("ProcessMessage() = %q, want %q", got, reactionFailure)
	}

	scores, _ := st.TopScores(ctx, 65, 10)
	if len(scores) != 0 {
		t.Errorf("TopScores() = %v, want empty since collect failed", scores)
	}
}

func TestProcessMessage_CustomMapDropped(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{err: puttday.ErrCustomMap}

	msg := IncomingMessage{Content: "https://putt.day/s/qSSLaHbOxWnI", AuthorID: "u1"}
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != "" {
		t.Errorf("ProcessMessage() = %q, want empty (no reaction) for a custom map", got)
	}

	exists, err := st.Exists(ctx, "https://putt.day/s/qSSLaHbOxWnI")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Errorf("Exists() = true, want false: a custom-map share should not be recorded")
	}
}

func TestProcessMessage_DuplicateStillReactsSuccess(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	collector := &fakeCollector{score: &puttday.SharedScore{Hole: 65, Strokes: 6, Link: "https://putt.day/s/abc"}}

	msg := IncomingMessage{Content: "https://putt.day/s/abc", AuthorID: "u1"}
	ProcessMessage(ctx, collector, st, msg, "")
	got := ProcessMessage(ctx, collector, st, msg, "")
	if got != reactionSuccess {
		t.Errorf("ProcessMessage() on duplicate = %q, want %q", got, reactionSuccess)
	}

	scores, _ := st.TopScores(ctx, 65, 10)
	if len(scores) != 1 {
		t.Errorf("TopScores() len = %d, want 1 (duplicate should not create a second record)", len(scores))
	}
}
