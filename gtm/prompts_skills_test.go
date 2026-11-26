package gtm

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/paolobietolini/gtm-mcp-server/skills"
)

// Prompts carry the skill files themselves, so guidance exists in one place.
func TestPromptsIncludeSkillFiles(t *testing.T) {
	request := func(args map[string]string) *mcp.GetPromptRequest {
		return &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Arguments: args}}
	}
	ctx := context.Background()
	const data = `{"tags": ["marker-100%"]}`
	for _, tc := range []struct {
		prompt string
		text   func(t *testing.T) string
		files  []string
		input  string
	}{
		{
			prompt: "audit_container",
			text: func(t *testing.T) string {
				text, err := auditPromptText(data)
				if err != nil {
					t.Fatal(err)
				}
				return text
			},
			files: []string{
				"gtm-container-audit/references/audit-checks.md",
				"gtm-safe-edit/references/naming-organization.md",
				"gtm-ga4-consent/references/ga4-consent.md",
			},
			input: data,
		},
		{
			prompt: "generate_tracking_plan",
			text: func(t *testing.T) string {
				text, err := trackingPlanPromptText(data)
				if err != nil {
					t.Fatal(err)
				}
				return text
			},
			files: []string{"gtm-tracking-plan/references/plan-template.md"},
			input: data,
		},
		{
			prompt: "suggest_ga4_setup",
			text: func(t *testing.T) string {
				res, err := handleSuggestGA4SetupPrompt(ctx, request(map[string]string{"goals": "marker-goals"}))
				return promptText(t, res, err)
			},
			files: []string{
				"gtm-ga4-consent/references/ga4-setup.md",
				"gtm-ga4-consent/references/ga4-consent.md",
				"gtm-safe-edit/references/naming-organization.md",
			},
			input: "marker-goals",
		},
		{
			prompt: "find_gallery_template",
			text: func(t *testing.T) string {
				res, err := handleFindGalleryTemplatePrompt(ctx, request(map[string]string{"templateName": "marker-name"}))
				return promptText(t, res, err)
			},
			files: []string{"gtm-safe-edit/references/gallery-templates.md"},
			input: "marker-name",
		},
	} {
		t.Run(tc.prompt, func(t *testing.T) {
			text := tc.text(t)
			if !strings.Contains(text, tc.input) {
				t.Errorf("prompt does not contain its input %q", tc.input)
			}
			for _, file := range tc.files {
				want, err := fs.ReadFile(skills.FS, file)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(text, string(want)) {
					t.Errorf("prompt does not include %s", file)
				}
			}
			if strings.Contains(text, "%!") {
				t.Error("prompt has a formatting error")
			}
		})
	}
}
