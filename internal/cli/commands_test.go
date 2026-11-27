package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

const ws = "accounts/1/containers/2/workspaces/3"

func TestDiscovery(t *testing.T) {
	h := newHarness(t).
		on("GET accounts", `{"account":[{"accountId":"1","name":"Acme"}]}`).
		on("GET accounts/1/containers", `{"container":[{"containerId":"2","name":"Web"}]}`).
		on("GET accounts/1/containers/2/workspaces", `{"workspace":[{"workspaceId":"3","name":"Default"}]}`).
		on("GET "+ws+"/status", `{"workspaceChange":[{},{}]}`)

	if got := h.ok("accounts", "list"); at(got["accounts"].([]any)[0], "name") != "Acme" {
		t.Fatalf("%v", got)
	}
	if got := h.ok("containers", "list", "--account", "1"); at(got["containers"].([]any)[0], "containerId") != "2" {
		t.Fatalf("%v", got)
	}
	if got := h.ok("workspaces", "list", "--account=1", "--container=2"); at(got["workspaces"].([]any)[0], "workspaceId") != "3" {
		t.Fatalf("%v", got)
	}
	got := h.ok("workspaces", "status", "--account", "1", "--container", "2", "--workspace", "3")
	if at(got, "status", "changeCount") != float64(2) || at(got, "status", "hasConflicts") != false {
		t.Fatalf("%v", got)
	}
}

func TestEmptyListIsAnArray(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/tags", `{}`)
	got := h.run("tags", "list")
	if got.exit != 0 || !strings.Contains(got.stdout, `"tags": []`) {
		t.Fatalf("%+v", got)
	}
}

func TestEntityCRUD(t *testing.T) {
	for _, tc := range []struct {
		noun, key, idFlag, create, update string
	}{
		{"tags", "tag", "--tag-id",
			`{"name":"New","type":"html","firingTriggerId":["7"],"parameter":[{"type":"template","key":"html","value":"<b></b>"}]}`,
			`{"name":"Renamed"}`},
		{"triggers", "trigger", "--trigger-id",
			`{"name":"New","type":"pageview"}`, `{"name":"Renamed","type":"pageview"}`},
		{"variables", "variable", "--variable-id",
			`{"name":"New","type":"c","parameter":[{"type":"template","key":"value","value":"x"}]}`,
			`{"name":"Renamed","type":"c"}`},
	} {
		t.Run(tc.noun, func(t *testing.T) {
			base := ws + "/" + tc.noun
			entity := `{"` + tc.key + `Id":"4","name":"Old","type":"html","notes":"keep","fingerprint":"fp"}`
			h := newHarness(t).withWorkspace().
				on("GET "+base, `{"`+tc.key+`":[`+entity+`]}`).
				on("GET "+base+"/4", entity).
				on("POST "+base, entity).
				on("PUT "+base+"/4", entity).
				on("DELETE "+base+"/4", ``)

			if got := h.ok(tc.noun, "list"); len(got[tc.noun].([]any)) != 1 {
				t.Fatalf("list: %v", got)
			}
			if got := h.ok(tc.noun, "get", tc.idFlag, "4"); at(got, tc.key, "notes") != "keep" {
				t.Fatalf("get: %v", got)
			}

			h.stdin = tc.create
			if got := h.ok(tc.noun, "create", "--file", "-"); got["success"] != true || at(got, tc.key, tc.key+"Id") != "4" {
				t.Fatalf("create: %v", got)
			}
			sent := h.sent()
			if body := decodeObject(t, sent[len(sent)-1].body); body["name"] != "New" {
				t.Fatalf("create body: %v", body)
			}

			file := filepath.Join(t.TempDir(), "update.json")
			if err := os.WriteFile(file, []byte(tc.update), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := h.ok(tc.noun, "update", tc.idFlag, "4", "--file", file); got["success"] != true {
				t.Fatalf("update: %v", got)
			}
			sent = h.sent()
			put := sent[len(sent)-1]
			if body := decodeObject(t, put.body); put.method != "PUT" || body["name"] != "Renamed" || body["notes"] != "keep" || !strings.Contains(put.query, "fingerprint=fp") {
				t.Fatalf("update request: %+v", put)
			}

			if got := h.ok(tc.noun, "delete", tc.idFlag, "4", "--confirm"); got["success"] != true {
				t.Fatalf("delete: %v", got)
			}
			sent = h.sent()
			if last := sent[len(sent)-1]; last.method != "DELETE" || last.path != base+"/4" {
				t.Fatalf("delete request: %+v", last)
			}
		})
	}
}

func TestVersionsCreateAndPublish(t *testing.T) {
	h := newHarness(t).withWorkspace().
		on("GET "+ws+"/status", `{"workspaceChange":[{}]}`).
		on("POST "+ws+":create_version", `{"containerVersion":{"containerVersionId":"9","name":"Release"}}`).
		on("POST accounts/1/containers/2/versions/9:publish", `{"containerVersion":{"containerVersionId":"9","name":"Release"}}`)

	got := h.ok("versions", "create", "--name", "Release", "--notes", "why")
	if at(got, "version", "containerVersionId") != "9" {
		t.Fatalf("create: %v", got)
	}
	sent := h.sent()
	if body := decodeObject(t, sent[1].body); body["name"] != "Release" || body["notes"] != "why" {
		t.Fatalf("create body: %v", body)
	}
	got = h.ok("versions", "publish", "--version-id", "9", "--confirm")
	if got["message"] != "Version 9 is now LIVE" {
		t.Fatalf("publish: %v", got)
	}
}

func TestVersionsCreateWithoutChanges(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/status", `{}`)
	message := h.fails(6, "precondition_failed", "versions", "create")
	if message != "No changes in workspace to create version from" || len(h.sent()) != 1 {
		t.Fatalf("message=%q requests=%+v", message, h.sent())
	}
}

const workspaceStatus = `{
  "workspaceChange": [
    {"changeStatus": "updated", "tag": {"tagId": "4", "name": "GA4 - purchase", "type": "gaawe", "fingerprint": "f2",
      "parameter": [{"type": "template", "key": "eventName", "value": "purchase"}]}},
    {"changeStatus": "added", "trigger": {"triggerId": "9", "name": "CE - purchase", "type": "customEvent"}},
    {"changeStatus": "deleted", "variable": {"variableId": "5", "name": "DLV - old", "type": "v"}}
  ],
  "mergeConflict": [
    {"entityInWorkspace": {"tag": {"tagId": "6", "name": "HTML - banner"}},
     "entityInBaseVersion": {"tag": {"tagId": "6", "name": "HTML - banner v2"}}},
    {"entityInBaseVersion": {"variable": {"variableId": "7", "name": "Const"}}}
  ]
}`

func TestWorkspaceStatusListsChanges(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/status", workspaceStatus)

	got := h.ok("workspaces", "status")
	changes, _ := at(got, "status", "changes").([]any)
	if len(changes) != 3 || at(got, "status", "changeCount") != float64(3) {
		t.Fatalf("changes: %v", got)
	}
	want := map[string]any{"type": "tag", "id": "4", "name": "GA4 - purchase", "changeStatus": "updated"}
	if !reflect.DeepEqual(changes[0], want) {
		t.Fatalf("change without --entities: %v", changes[0])
	}
	if at(changes[1], "changeStatus") != "added" || at(changes[2], "type") != "variable" || at(changes[2], "changeStatus") != "deleted" {
		t.Fatalf("changes: %v", changes)
	}
	conflicts, _ := at(got, "status", "conflicts").([]any)
	// A conflict whose workspace side was deleted is named from the base version.
	if len(conflicts) != 2 || at(conflicts[0], "name") != "HTML - banner" ||
		at(conflicts[1], "type") != "variable" || at(conflicts[1], "id") != "7" || at(conflicts[1], "baseEntity") != nil {
		t.Fatalf("conflicts: %v", conflicts)
	}

	got = h.ok("workspaces", "status", "--entities")
	changes, _ = at(got, "status", "changes").([]any)
	params, _ := at(changes[0], "entity", "parameter").([]any)
	if len(params) != 1 || at(params[0], "value") != "purchase" || at(changes[0], "entity", "fingerprint") != "f2" {
		t.Fatalf("change with --entities: %v", changes[0])
	}
	conflicts, _ = at(got, "status", "conflicts").([]any)
	if at(conflicts[0], "entity", "name") != "HTML - banner" || at(conflicts[0], "baseEntity", "name") != "HTML - banner v2" ||
		at(conflicts[1], "entity") != nil || at(conflicts[1], "baseEntity", "name") != "Const" {
		t.Fatalf("conflicts with --entities: %v", conflicts)
	}
}

func TestWorkspaceStatusWithoutChangesHasEmptyLists(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/status", `{}`)
	got := h.run("workspaces", "status")
	if got.exit != 0 || !strings.Contains(got.stdout, `"changes": []`) || !strings.Contains(got.stdout, `"conflicts": []`) {
		t.Fatalf("%+v", got)
	}
}

func TestVersionsGet(t *testing.T) {
	const container = "accounts/1/containers/2"
	version := `{"containerVersionId":"8","name":"Release","tag":[{"tagId":"4","name":"GA4 - purchase"}]}`
	h := newHarness(t).withWorkspace().
		on("GET "+container+"/versions/8", version).
		on("GET "+container+"/versions:live", version).
		on("GET "+container+"/version_headers:latest", `{"containerVersionId":"8"}`)

	for _, args := range [][]string{{"--version-id", "8"}, {"--live"}, {"--latest"}} {
		got := h.ok(append([]string{"versions", "get"}, args...)...)
		tags, _ := at(got, "version", "tag").([]any)
		if at(got, "version", "containerVersionId") != "8" || len(tags) != 1 || at(tags[0], "name") != "GA4 - purchase" {
			t.Fatalf("versions get %v: %v", args, got)
		}
	}
	var paths []string
	for _, r := range h.sent() {
		paths = append(paths, r.method+" "+r.path)
	}
	want := []string{
		"GET " + container + "/versions/8",
		"GET " + container + "/versions:live",
		"GET " + container + "/version_headers:latest",
		"GET " + container + "/versions/8",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("requests: %v", paths)
	}
}

func TestVersionsGetNeedsOneVersion(t *testing.T) {
	h := newHarness(t).withWorkspace()
	h.fails(2, "usage", "versions", "get")
	h.fails(2, "usage", "versions", "get", "--latest", "--live")
	h.fails(2, "usage", "versions", "get", "--version-id", "8", "--latest")
	h.fails(2, "usage", "versions", "get", "--version-id", "8/../9")
	h.noRequests()
}

func TestDestructiveCommandsRequireConfirm(t *testing.T) {
	for _, args := range [][]string{
		{"tags", "delete", "--tag-id", "4"},
		{"triggers", "delete", "--trigger-id", "4"},
		{"variables", "delete", "--variable-id", "4"},
		{"versions", "publish", "--version-id", "9"},
		{"versions", "publish", "--version-id", "9", "--confirm=false"},
		{"tags", "delete", "--tag-id", "4", "--dry-run=false"},
	} {
		h := newHarness(t).withWorkspace()
		message := h.fails(8, "confirmation_required", args...)
		if !strings.Contains(message, "--confirm") || strings.Contains(message, "confirm: true") {
			t.Errorf("message should name the flag: %q", message)
		}
		h.noRequests()
	}
}

// "--confirm false" must not be read as a confirmation followed by noise.
func TestStrayArgumentsAreRejected(t *testing.T) {
	for _, args := range [][]string{
		{"tags", "delete", "--tag-id", "4", "--confirm", "false"},
		{"tags", "delete", "4", "--confirm"},
		{"tags", "list", "extra"},
	} {
		h := newHarness(t).withWorkspace()
		h.fails(2, "usage", args...)
		h.noRequests()
	}
}

// IDs are placed in URL paths, so anything but digits could address another resource.
func TestIDsMustBeNumeric(t *testing.T) {
	for _, args := range [][]string{
		{"tags", "delete", "--tag-id", "4/../5", "--confirm"},
		{"tags", "get", "--tag-id", "4", "--workspace", "3/tags/9"},
		{"workspaces", "list", "--container", "GTM-ABC123"},
		{"versions", "publish", "--version-id", "../9", "--confirm"},
	} {
		h := newHarness(t).withWorkspace()
		h.fails(2, "usage", args...)
		h.noRequests()
	}
}

func TestMissingIDs(t *testing.T) {
	h := newHarness(t)
	if message := h.fails(2, "usage", "tags", "list", "--account", "1"); !strings.Contains(message, "--container") || !strings.Contains(message, "GTM_CONTAINER_ID") {
		t.Fatalf("%q", message)
	}
	h.withWorkspace().fails(2, "usage", "tags", "get")
	h.withWorkspace().fails(2, "usage", "tags", "create")
	h.noRequests()
}

func TestAPIErrorsMapToExitCodes(t *testing.T) {
	for _, tc := range []struct {
		status        int
		body          string
		exit          int
		code, mention string
	}{
		{400, `{"error":{"code":400,"message":"bad"}}`, 2, "invalid_request", "bad"},
		{401, `{"error":{"code":401,"message":"expired"}}`, 3, "auth", "expired"},
		{403, `{"error":{"code":403,"message":"denied"}}`, 4, "permission_denied", "denied"},
		{403, `{"error":{"code":403,"message":"Request had insufficient authentication scopes."}}`, 3, "auth", "scopes"},
		{404, `{"error":{"code":404,"message":"gone"}}`, 5, "not_found", "gone"},
		{409, `{"error":{"code":409,"message":"stale"}}`, 6, "conflict", "stale"},
		{429, `{"error":{"code":429,"message":"slow down"}}`, 7, "rate_limited", "slow down"},
		{500, `{"error":{"code":500,"message":"boom"}}`, 1, "api_error", "boom"},
	} {
		// A delete is never retried, so each status is seen exactly once.
		h := newHarness(t).withWorkspace().onStatus("DELETE "+ws+"/tags/4", tc.status, tc.body)
		message := h.fails(tc.exit, tc.code, "tags", "delete", "--tag-id", "4", "--confirm")
		if !strings.Contains(message, tc.mention) {
			t.Errorf("%d: message %q", tc.status, message)
		}
	}
}

func TestHelpAndUsage(t *testing.T) {
	h := newHarness(t).noCredentials()
	for _, args := range [][]string{{}, {"help"}, {"--help"}, {"tags"}, {"tags", "--help"}, {"tags", "create", "--help"}} {
		got := h.run(args...)
		if got.exit != 0 || !strings.Contains(got.stdout, "gtm-cli") || got.stderr != "" {
			t.Errorf("%v: %+v", args, got)
		}
	}
	if got := h.run("tags", "create", "-h"); !strings.Contains(got.stdout, "-dry-run") {
		t.Errorf("flag help missing: %s", got.stdout)
	}
	for _, args := range [][]string{
		{"nope"}, {"tags", "nope"}, {"tags", "list", "--nope"}, {"tags", "list", "--dry-run"},
		{"--output", "table", "tags", "list"}, {"accounts", "list", "--output", "yaml"},
	} {
		h.fails(2, "usage", args...)
	}
	h.noRequests()
}

func TestVersionCommand(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		if got := newHarness(t).ok(args...); got["version"] != "1.2.3" {
			t.Fatalf("%v", got)
		}
	}
}

// The command is gtm-cli so that it does not collide with other programs named
// gtm. Nothing it prints may tell people to run the old name.
func TestMessagesUseTheCommandName(t *testing.T) {
	oldName := regexp.MustCompile("(^|[\\s`])gtm([\\s`]|$)")
	runs := [][]string{{"help"}, {"nonsense"}, {"tags", "nonsense"}, {"--output", "table"}}
	for _, cmd := range commands {
		runs = append(runs, []string{cmd.noun, cmd.verb, "--help"})
	}
	for _, args := range runs {
		got := newHarness(t).run(args...)
		printed := got.stdout + got.stderr
		if !strings.Contains(printed, "gtm-cli") || oldName.MatchString(printed) {
			t.Errorf("gtm-cli %s should name the command gtm-cli:\n%s", strings.Join(args, " "), printed)
		}
	}
	if oldName.MatchString(authHint) || !strings.Contains(authHint, "gtm-cli auth status") {
		t.Errorf("auth hint names the wrong command: %s", authHint)
	}
}
