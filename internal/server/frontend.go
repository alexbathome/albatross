package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Frontend serves a built single-page frontend from dist, falling back to
// index.html for paths that aren't built assets so client-side routes
// (e.g. /holes/66) survive a full page load. Requests under /api/ are never
// served the app shell: an unmatched API path stays a 404, not an HTML page.
//
// If dist holds no build (no index.html), every request 404s and the server
// effectively serves only the API.
func Frontend(dist fs.FS) http.Handler {
	fileServer := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			if _, err := fs.Stat(dist, "index.html"); err != nil {
				http.NotFound(w, r)
				return
			}
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
