//go:build prod

// Package webapp embeds the built React web UI (playground + IDE) so production
// binaries can serve it without an external directory. Build the app first
// (make web-build) and compile with the `prod` tag (make build-prod).
package webapp

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns the embedded built web app rooted at its dist directory, and
// true when assets are available.
func Assets() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	// Guard against an empty embed (no index.html) so callers can fall back.
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
