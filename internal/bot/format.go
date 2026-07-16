package bot

import (
	"fmt"
	"strings"

	"github.com/alexbathome/albatross/pkg/store"
)

// FormatTopScores renders the /top leaderboard reply.
func FormatTopScores(hole int, requestedCount int, scores []store.ScoreRecord) string {
	header := fmt.Sprintf("Scores for Top %d on Hole %d", requestedCount, hole)
	if len(scores) == 0 {
		return header + "\nNo scores recorded yet."
	}
	var b strings.Builder
	b.WriteString(header)
	for _, rec := range scores {
		fmt.Fprintf(&b, "\n<@%s> %d %s", rec.UserID, rec.Strokes, rec.ShareLink)
	}
	return b.String()
}

// FormatUserScores renders the /score reply for a single user on a hole.
func FormatUserScores(hole int, userID string, scores []store.ScoreRecord) string {
	if len(scores) == 0 {
		return fmt.Sprintf("<@%s> has no recorded scores on Hole %d.", userID, hole)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Scores for <@%s> on Hole %d", userID, hole)
	for _, rec := range scores {
		fmt.Fprintf(&b, "\n%d %s", rec.Strokes, rec.ShareLink)
	}
	return b.String()
}
