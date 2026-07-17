package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/alexbathome/albatross/internal/server"
	apitypes "github.com/alexbathome/albatross/pkg/api"
	"github.com/alexbathome/albatross/pkg/store"
)

const (
	defaultTopLimit  = 10
	maxTopLimit      = 25 // mirrors the bot's /top command cap
	defaultUserLimit = 25 // mirrors the bot's /score command
)

// handleTopScores returns the leaderboard for a hole.
//
//	@Summary		Leaderboard for a hole
//	@Description	Returns up to limit scores for hole, ascending by strokes (lowest first).
//	@Tags			scores
//	@Produce		json
//	@Param			hole	path		int	true	"Hole number"
//	@Param			limit	query		int	false	"Max results (default 10, max 25)"
//	@Success		200		{array}		apitypes.ScoreRecord
//	@Failure		400		{object}	apitypes.ErrorResponse
//	@Failure		500		{object}	apitypes.ErrorResponse
//	@Router			/holes/{hole}/top [get]
func (a *API) handleTopScores(s *server.Server, w http.ResponseWriter, r *http.Request) {
	hole, err := requiredIntPath(r, "hole")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := clampInt(optionalIntQuery(r, "limit", defaultTopLimit), 1, maxTopLimit)

	recs, err := s.Store().TopScores(r.Context(), hole, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "looking up top scores failed")
		return
	}
	writeJSON(w, http.StatusOK, toAPIScoreRecords(recs))
}

// handleUserScores returns a user's recorded plays on a hole.
//
//	@Summary		A user's scores for a hole
//	@Description	Returns up to limit of userID's records for hole, ascending by strokes. Not deduplicated: every recorded play is returned.
//	@Tags			scores
//	@Produce		json
//	@Param			userID	path		string	true	"Discord user ID"
//	@Param			hole	path		int		true	"Hole number"
//	@Param			limit	query		int		false	"Max results (max 25)"
//	@Success		200		{array}		apitypes.ScoreRecord
//	@Failure		400		{object}	apitypes.ErrorResponse
//	@Failure		500		{object}	apitypes.ErrorResponse
//	@Router			/users/{userID}/holes/{hole} [get]
func (a *API) handleUserScores(s *server.Server, w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	hole, err := requiredIntPath(r, "hole")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := clampInt(optionalIntQuery(r, "limit", defaultUserLimit), 1, defaultUserLimit)

	recs, err := s.Store().UserScores(r.Context(), hole, userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "looking up user scores failed")
		return
	}
	writeJSON(w, http.StatusOK, toAPIScoreRecords(recs))
}

func toAPIScoreRecords(recs []store.ScoreRecord) []apitypes.ScoreRecord {
	out := make([]apitypes.ScoreRecord, len(recs))
	for i, rec := range recs {
		out[i] = apitypes.ScoreRecord{
			ShareLink:  rec.ShareLink,
			Hole:       rec.Hole,
			Strokes:    rec.Strokes,
			UserID:     rec.UserID,
			GuildID:    rec.GuildID,
			ChannelID:  rec.ChannelID,
			MessageID:  rec.MessageID,
			RecordedAt: rec.RecordedAt,
		}
	}
	return out
}

func requiredIntPath(r *http.Request, key string) (int, error) {
	v, err := strconv.Atoi(r.PathValue(key))
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}
	return v, nil
}

func optionalIntQuery(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

func clampInt(v, minimum, maximum int) int {
	if v < minimum {
		return minimum
	}
	if v > maximum {
		return maximum
	}
	return v
}
