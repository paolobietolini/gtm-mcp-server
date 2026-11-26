package gtm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func promptText(t *testing.T, res *mcp.GetPromptResult, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return res.Messages[0].Content.(*mcp.TextContent).Text
}

// GTM saves gaawc as googtag and eventParameters as eventSettingsTable, so the
// examples must show the stored shapes.
func TestTagTemplatesUseCurrentShapes(t *testing.T) {
	var googleTag bool
	for _, tmpl := range GetTagTemplates() {
		var params []Parameter
		if err := json.Unmarshal([]byte(tmpl.Parameters), &params); err != nil {
			t.Errorf("%s: parameters are not valid JSON: %v", tmpl.Name, err)
		}
		text := tmpl.Type + tmpl.Parameters + tmpl.Notes
		for _, legacy := range []string{"gaawc", `"eventParameters"`} {
			if strings.Contains(text, legacy) {
				t.Errorf("%s: uses legacy %s", tmpl.Name, legacy)
			}
		}
		if tmpl.Type == "googtag" {
			googleTag = len(params) == 1 && params[0].Key == "tagId"
		}
	}
	if !googleTag {
		t.Error("no googtag example with a tagId parameter")
	}
}

func TestSuggestGA4SetupPromptFollowsNamingConvention(t *testing.T) {
	req := &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Arguments: map[string]string{"goals": "purchases"}}}
	res, err := handleSuggestGA4SetupPrompt(context.Background(), req)
	text := promptText(t, res, err)
	if !strings.Contains(text, "<Platform> - <Type> - <Descriptor>") || strings.Contains(text, "[Category] - [Action]") {
		t.Error("prompt must use the naming convention from naming-organization")
	}
}

func TestFindGalleryTemplatePromptExamples(t *testing.T) {
	req := &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Arguments: map[string]string{"templateName": "cookiebot"}}}
	res, err := handleFindGalleryTemplatePrompt(context.Background(), req)
	text := promptText(t, res, err)
	for _, want := range []string{
		"| iubenda | gtm-cookie-solution |",
		"| Usercentrics | gtm-templates-cookiebot-cmp |",
		"| meta | GoogleTagManager-WebTemplate-For-FacebookPixel |",
		"cvt_{galleryTemplateId}",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt text missing %q", want)
		}
	}
	for _, bad := range []string{"nicktue-gtm-templates", "cvt_{containerId}_{templateId}"} {
		if strings.Contains(text, bad) {
			t.Errorf("prompt text still contains %q", bad)
		}
	}
}
