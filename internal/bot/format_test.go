package bot

import (
	"testing"

	"github.com/alexbathome/albatross/pkg/store"
)

func TestFormatTopScores_Empty(t *testing.T) {
	got := FormatTopScores(65, 10, nil)
	want := "Scores for Top 10 on Hole 65\nNo scores recorded yet."
	if got != want {
		t.Errorf("FormatTopScores() = %q, want %q", got, want)
	}
}

func TestFormatTopScores(t *testing.T) {
	scores := []store.ScoreRecord{
		{UserID: "u1", Strokes: 5, ShareLink: "https://putt.day/s/abc"},
		{UserID: "u2", Strokes: 6, ShareLink: "https://putt.day/s/def"},
	}
	got := FormatTopScores(65, 10, scores)
	want := "Scores for Top 10 on Hole 65\n" +
		"<@u1> 5 https://putt.day/s/abc\n" +
		"<@u2> 6 https://putt.day/s/def"
	if got != want {
		t.Errorf("FormatTopScores() = %q, want %q", got, want)
	}
}

func TestFormatTopScores_RequestedCountShownRegardlessOfRows(t *testing.T) {
	scores := []store.ScoreRecord{
		{UserID: "u1", Strokes: 5, ShareLink: "https://putt.day/s/abc"},
	}
	got := FormatTopScores(65, 10, scores)
	want := "Scores for Top 10 on Hole 65\n<@u1> 5 https://putt.day/s/abc"
	if got != want {
		t.Errorf("FormatTopScores() = %q, want %q", got, want)
	}
}

func TestFormatUserScores_Empty(t *testing.T) {
	got := FormatUserScores(65, "u1", nil)
	want := "<@u1> has no recorded scores on Hole 65."
	if got != want {
		t.Errorf("FormatUserScores() = %q, want %q", got, want)
	}
}

func TestFormatUserScores(t *testing.T) {
	scores := []store.ScoreRecord{
		{Strokes: 5, ShareLink: "https://putt.day/s/abc"},
		{Strokes: 7, ShareLink: "https://putt.day/s/def"},
	}
	got := FormatUserScores(65, "u1", scores)
	want := "Scores for <@u1> on Hole 65\n5 https://putt.day/s/abc\n7 https://putt.day/s/def"
	if got != want {
		t.Errorf("FormatUserScores() = %q, want %q", got, want)
	}
}
