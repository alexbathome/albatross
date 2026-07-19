package api

import (
	"net/http"

	"github.com/alexbathome/albatross/internal/server"
	apitypes "github.com/alexbathome/albatross/pkg/api"
	"github.com/alexbathome/albatross/pkg/store"
)

const (
	defaultHolesLimit = 20
	// maxHolesLimit is high enough that a client can request the full
	// history of holes (the web UI's "down to 1" archive list) in one call
	// rather than paginating — hole numbers are small sequential ints, so
	// even years of daily holes comfortably fit one response.
	maxHolesLimit = 1000
)

// handleListHoles returns the most recently registered holes, descending by
// hole number.
//
//	@Summary		List recent holes
//	@Description	Returns up to limit registered holes, descending by hole number.
//	@Tags			holes
//	@Produce		json
//	@Param			limit	query		int	false	"Max results (default 20, max 1000)"
//	@Success		200		{array}		apitypes.Hole
//	@Failure		500		{object}	apitypes.ErrorResponse
//	@Router			/holes [get]
func (a *API) handleListHoles(s *server.Server, w http.ResponseWriter, r *http.Request) {
	limit := clampInt(optionalIntQuery(r, "limit", defaultHolesLimit), 1, maxHolesLimit)

	holes, err := s.Store().ListHoles(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "looking up holes failed")
		return
	}
	writeJSON(w, http.StatusOK, toAPIHoles(holes))
}

func toAPIHoles(holes []store.Hole) []apitypes.Hole {
	out := make([]apitypes.Hole, len(holes))
	for i, h := range holes {
		out[i] = apitypes.Hole{
			Number:     h.Number,
			Custom:     h.Custom,
			TopStrokes: h.TopStrokes,
		}
	}
	return out
}