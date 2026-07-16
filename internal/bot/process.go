package bot

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/alexbathome/albatross/pkg/puttday"
	"github.com/alexbathome/albatross/pkg/store"
)

// Collector abstracts *puttday.collector (an unexported concrete type),
// letting this package depend on it structurally and tests substitute a fake.
type Collector interface {
	Collect(ctx context.Context, shareLink string) (*puttday.SharedScore, error)
}

// IncomingMessage is a discordgo-independent projection of the fields
// ProcessMessage needs from a *discordgo.MessageCreate.
type IncomingMessage struct {
	Content   string
	IsBot     bool
	AuthorID  string
	GuildID   string
	ChannelID string
	MessageID string
}

const (
	reactionSuccess = "⛳"
	reactionFailure = "⚠️"
)

// ProcessMessage inspects msg for a putt.day share link and, if found, scrapes
// and persists it. Returns the reaction emoji to apply, or "" if msg had no
// share link at all, or the link was for a custom map (no reaction in either
// case).
//
// If allowedGuildID is non-empty, messages from any other guild (or from DMs)
// are dropped without reaction: the bot only ever collects scores for the one
// guild it is configured to serve. An empty allowedGuildID allows all guilds.
func ProcessMessage(ctx context.Context, collector Collector, st store.Store, msg IncomingMessage, allowedGuildID string) string {
	if msg.IsBot {
		return ""
	}
	if allowedGuildID != "" && msg.GuildID != allowedGuildID {
		return ""
	}
	link, ok := puttday.ExtractShareLink(msg.Content)
	if !ok {
		return ""
	}

	score, err := collector.Collect(ctx, link)
	if errors.Is(err, puttday.ErrCustomMap) {
		return ""
	}
	if err != nil {
		log.Printf("puttday collect failed for %s: %v", link, err)
		return reactionFailure
	}

	rec := store.ScoreRecord{
		ShareLink:  score.Link,
		Hole:       score.Hole,
		Strokes:    score.Strokes,
		UserID:     msg.AuthorID,
		GuildID:    msg.GuildID,
		ChannelID:  msg.ChannelID,
		MessageID:  msg.MessageID,
		RecordedAt: time.Now().UTC(),
	}
	if _, err := st.SaveScore(ctx, rec); err != nil {
		log.Printf("save score failed for %s: %v", link, err)
		return reactionFailure
	}
	return reactionSuccess
}
