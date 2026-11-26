// Package skills embeds the Agent Skills in this directory, so the server
// serves the same files that people install. Each folder is one skill and
// refers only to its own files.
package skills

import "embed"

// FS holds every skill folder at its root, named after the skill.
//
//go:embed gtm-*
var FS embed.FS
