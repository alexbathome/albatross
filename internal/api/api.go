package api

import (
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/alexbathome/albatross/docs" //nolint:revive // registers the generated swagger spec
	"github.com/alexbathome/albatross/internal/server"
)

// API registers the albatross HTTP REST API's routes onto a server.Server.
type API struct {
	server *server.Server
}

// NewAPI returns an API that dispatches to s's underlying store.
func NewAPI(s *server.Server) *API {
	return &API{
		server: s,
	}
}

// RegisterRoutes wires every route, including the swagger UI, onto the
// underlying server.
func (a *API) RegisterRoutes() {
	a.server.Register("GET /api/holes", a.handleListHoles)
	a.server.Register("GET /api/holes/{hole}/top", a.handleTopScores)
	a.server.Register("GET /api/scores", a.handleSearchScores)
	a.server.Register("GET /api/users/{userID}/holes/{hole}", a.handleUserScores)

	a.server.Handle("/swagger/", httpSwagger.WrapHandler)
}
