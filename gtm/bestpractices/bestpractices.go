// Package bestpractices provides the GTM configuration best-practice documents
// served via MCP resources and injected into prompts. The documents are files of
// the embedded Agent Skills; only the index of resource URIs lives here.
package bestpractices

import (
	_ "embed"
	"fmt"
	"maps"
	"slices"
	"strings"

	"gtm-mcp-server/skills"
)

//go:embed docs/index.md
var index string

const indexTopic = "index"

// skillDocs maps each topic to its canonical file in the embedded skills.
var skillDocs = map[string]string{
	"ga4-consent":         "gtm-ga4-consent/references/ga4-consent.md",
	"naming-organization": "gtm-safe-edit/references/naming-organization.md",
	"safe-edit-workflow":  "gtm-safe-edit/references/safe-edit-workflow.md",
	"server-side":         "gtm-server-side/references/server-side.md",
	"tool-input-formats":  "gtm-mcp/references/tool-input-formats.md",
}

// Topics returns the available best-practice topic names.
func Topics() []string {
	return slices.Sorted(slices.Values(append(slices.Collect(maps.Keys(skillDocs)), indexTopic)))
}

// Get returns the markdown content for a topic.
func Get(topic string) (string, error) {
	if topic == indexTopic {
		return index, nil
	}
	if path, ok := skillDocs[topic]; ok {
		data, err := skills.FS.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read best-practices topic %q: %w", topic, err)
		}
		return string(data), nil
	}
	return "", fmt.Errorf("unknown best-practices topic %q; valid topics: %s", topic, strings.Join(Topics(), ", "))
}
