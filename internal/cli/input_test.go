package cli

import (
	"strings"
	"testing"
)

const currentTag = `{"tagId":"4","name":"Old","type":"html","notes":"keep","paused":true,"fingerprint":"fp",
"firingTriggerId":["7"],"parameter":[{"type":"template","key":"html","value":"x"}],
"setupTag":[{"tagName":"a"}],"consentSettings":{"consentStatus":"notNeeded"}}`

func updateTag(t *testing.T, input string) map[string]any {
	t.Helper()
	h := newHarness(t).withWorkspace().
		on("GET "+ws+"/tags/4", currentTag).
		on("PUT "+ws+"/tags/4", currentTag)
	h.stdin = input
	h.ok("tags", "update", "--tag-id", "4", "--file", "-")
	sent := h.sent()
	return decodeObject(t, sent[len(sent)-1].body)
}

func TestTagUpdateOmittedKeysArePreserved(t *testing.T) {
	body := updateTag(t, `{}`)
	for _, key := range []string{"name", "type", "notes", "paused", "firingTriggerId", "parameter", "setupTag", "consentSettings"} {
		if body[key] == nil {
			t.Errorf("%s was lost: %v", key, body)
		}
	}
}

func TestTagUpdateExplicitValuesClear(t *testing.T) {
	body := updateTag(t, `{"notes":"","paused":false,"parameter":[],"setupTag":[]}`)
	for _, key := range []string{"paused", "parameter", "setupTag"} {
		if body[key] != nil {
			t.Errorf("%s was not cleared: %v", key, body[key])
		}
	}
	if body["notes"] != "" {
		t.Errorf("notes was not cleared: %v", body["notes"])
	}
}

// encoding/json accepts any casing, so presence detection must too.
func TestTagUpdateKeyCasingKeepsPresence(t *testing.T) {
	body := updateTag(t, `{"Notes":"","Paused":false}`)
	if body["notes"] != "" || body["paused"] != nil {
		t.Fatalf("%v", body)
	}
}

func TestTagUpdateNullPreserves(t *testing.T) {
	body := updateTag(t, `{"notes":null,"paused":null,"parameter":null,"setupTag":null}`)
	if body["notes"] != "keep" || body["paused"] != true || body["parameter"] == nil || body["setupTag"] == nil {
		t.Fatalf("%v", body)
	}
}

func TestTagUpdateConsent(t *testing.T) {
	body := updateTag(t, `{"consentStatus":"needed","consentTypes":["ad_storage"]}`)
	if at(body, "consentSettings", "consentStatus") != "needed" || at(body, "consentSettings", "consentType") == nil {
		t.Fatalf("%v", body)
	}
}

func TestTriggerUpdateRemapsClickFilter(t *testing.T) {
	h := newHarness(t).withWorkspace().
		on("GET "+ws+"/triggers/4", `{"triggerId":"4","name":"Old","type":"linkClick","fingerprint":"fp"}`).
		on("PUT "+ws+"/triggers/4", `{"triggerId":"4"}`)
	h.stdin = `{"name":"Click","type":"linkClick",
		"filter":[{"type":"equals","parameter":[]}],
		"autoEventFilter":[{"type":"contains","parameter":[]}]}`
	got := h.ok("triggers", "update", "--trigger-id", "4", "--file", "-")
	if message, _ := got["message"].(string); !strings.Contains(message, "automatically remapped") {
		t.Fatalf("no warning: %v", got)
	}
	sent := h.sent()
	body := decodeObject(t, sent[1].body)
	if filter, _ := body["filter"].([]any); len(filter) != 2 || body["autoEventFilter"] != nil {
		t.Fatalf("%v", body)
	}
}

func TestInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name, mention string
		args          []string
		input         string
	}{
		{"not JSON", "invalid", []string{"tags", "create"}, `name: x`},
		{"array", "JSON object", []string{"tags", "create"}, `[{}]`},
		{"null", "JSON object", []string{"tags", "create"}, `null`},
		{"empty", "invalid", []string{"tags", "create"}, ``},
		{"two documents", "after", []string{"tags", "create"}, `{"name":"a","type":"html","firingTriggerId":["1"]} {}`},
		{"unknown field", "firingTriggerIds", []string{"tags", "create"}, `{"name":"a","type":"html","firingTriggerIds":["1"]}`},
		{"unknown nested field", "values", []string{"tags", "create"}, `{"name":"a","type":"html","firingTriggerId":["1"],"parameter":[{"values":"x"}]}`},
		{"internal field", "HasNotes", []string{"tags", "create"}, `{"HasNotes":true}`},
		{"tag without trigger", "firing trigger", []string{"tags", "create"}, `{"name":"a","type":"html"}`},
		{"trigger without type", "trigger type", []string{"triggers", "create"}, `{"name":"a"}`},
		{"variable without name", "variable name", []string{"variables", "create"}, `{"type":"c"}`},
		{"trigger update without name", "trigger name", []string{"triggers", "update", "--trigger-id", "4"}, `{"type":"pageview"}`},
		{"variable update without type", "variable type", []string{"variables", "update", "--variable-id", "4"}, `{"name":"a"}`},
		{"consent types alone", "consentStatus", []string{"tags", "update", "--tag-id", "4"}, `{"consentTypes":["ad_storage"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t).withWorkspace()
			h.stdin = tc.input
			message := h.fails(2, "invalid_input", append(tc.args, "--file", "-")...)
			if !strings.Contains(message, tc.mention) {
				t.Errorf("message %q does not mention %q", message, tc.mention)
			}
			h.noRequests()
		})
	}
}

func TestUnreadableFile(t *testing.T) {
	h := newHarness(t).withWorkspace()
	h.fails(2, "invalid_input", "tags", "create", "--file", t.TempDir()+"/missing.json")
}
