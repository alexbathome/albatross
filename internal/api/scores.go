package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexbathome/albatross/internal/server"
	apitypes "github.com/alexbathome/albatross/pkg/api"
	"github.com/alexbathome/albatross/pkg/store"
)

const (
	defaultTopLimit    = 10
	maxTopLimit        = 25 // mirrors the bot's /top command cap
	defaultUserLimit   = 25 // mirrors the bot's /score command
	defaultSearchLimit = 50
	maxSearchLimit     = 100
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
	limit := limitQuery(r, defaultTopLimit, maxTopLimit)

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
	limit := limitQuery(r, defaultUserLimit, defaultUserLimit)

	recs, err := s.Store().UserScores(r.Context(), hole, userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "looking up user scores failed")
		return
	}
	writeJSON(w, http.StatusOK, toAPIScoreRecords(recs))
}

// handleSearchScores returns the best recorded play per hole for each user
// whose username matches the query.
//
//	@Summary		Search scores by username
//	@Description	Returns each matching user's best (lowest-stroke) play per hole, holes descending. Matches case-insensitively against the username recorded with each score.
//	@Tags			scores
//	@Produce		json
//	@Param			username	query		string	true	"Username to search for (substring match)"
//	@Param			limit		query		int		false	"Max results (default 50, max 100)"
//	@Success		200			{array}		apitypes.ScoreRecord
//	@Failure		400			{object}	apitypes.ErrorResponse
//	@Failure		500			{object}	apitypes.ErrorResponse
//	@Router			/scores [get]
func (a *API) handleSearchScores(s *server.Server, w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	limit := limitQuery(r, defaultSearchLimit, maxSearchLimit)

	recs, err := s.Store().SearchScores(r.Context(), username, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "searching scores failed")
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
			Username:   rec.Username,
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

// limitQuery returns the "limit" query parameter clamped to [1, maximum],
// or def if the parameter is absent or not an integer.
func limitQuery(r *http.Request, def, maximum int) int {
	v := def
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			v = parsed
		}
	}
	return min(max(v, 1), maximum)
}
