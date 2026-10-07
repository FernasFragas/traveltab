// Package writerdata embeds the allowlist and reviewed matching overrides.
package writerdata

import "embed"

//go:embed sources.json overrides.json
var Files embed.FS
