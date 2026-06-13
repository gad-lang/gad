//go:build !prod

// Package webapp provides access to the built web UI. In non-production builds
// no assets are embedded, so `gad ide` serves its bundled UI (or --static).
package webapp

import "io/fs"

// Assets returns no embedded assets in non-production builds.
func Assets() (fs.FS, bool) { return nil, false }
