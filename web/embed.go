// Package web embeds the built frontend so the server ships as a single
// binary. Build the frontend first (see README.md) — if dist/ holds no build,
// the binary still compiles and the server simply serves only the API.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the built frontend rooted at its index.html.
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		// Unreachable: "dist" is a valid path embedded above.
		panic(err)
	}
	return sub
}
