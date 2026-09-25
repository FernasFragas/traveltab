// Package guidedata embeds guides/guides.json in the web binary, so reviewed city intros are
// served from memory and need no file beside the executable. The package name differs from its
// directory on purpose: internal/guides is the code that reads and writes this file.
package guidedata

import _ "embed"

// JSON is the contents of guides.json, exactly as committed.
//
//go:embed guides.json
var JSON []byte
