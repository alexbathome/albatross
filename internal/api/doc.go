// Package api implements the albatross HTTP REST API: a thin JSON layer
// over [store.Store] for recording and querying putt.day scores, alongside
// the Discord bot.
//
// Run `go generate ./...` after changing any handler's swaggo annotations
// to regenerate the ../../docs package.
//
//	@title			Albatross API
//	@version		1.0
//	@description	Read-only REST API for querying putt.day scores tracked by the albatross Discord bot.
//	@description	Recording and removing scores is only done via the Discord bot itself; this API has no
//	@description	authentication, so it intentionally exposes no write or delete endpoints.
//
//	@license.name	MIT
//	@license.url	https://github.com/alexbathome/albatross/blob/main/LICENSE
//
//	@BasePath	/api
//
//go:generate go tool swag init -g doc.go -o ../../docs --parseInternal --parseDependency
package api
