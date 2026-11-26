package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/paolobietolini/gtm-mcp-server/auth"

	"golang.org/x/oauth2"
)

func TestDryRunCreateSendsNothingAndNeedsNoCredentials(t *testing.T) {
	h := newHarness(t).withWorkspace().noCredentials()
	h.stdin = `{"name":"New","type":"c"}`
	got := h.ok("variables", "create", "--file", "-", "--dry-run")
	if got["dryRun"] != true || at(got, "request", "method") != "POST" ||
		!strings.HasSuffix(at(got, "request", "url").(string), "/tagmanager/v2/"+ws+"/variables") ||
		at(got, "request", "body", "name") != "New" {
		t.Fatalf("%v", got)
	}
	h.noRequests()
}

func TestDryRunUpdateReadsButDoesNotWrite(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/tags/4", currentTag)
	h.stdin = `{"name":"Renamed"}`
	got := h.ok("tags", "update", "--tag-id", "4", "--file", "-", "--dry-run")
	if at(got, "request", "method") != "PUT" || at(got, "request", "query", "fingerprint") != "fp" ||
		at(got, "request", "body", "name") != "Renamed" || at(got, "request", "body", "notes") != "keep" {
		t.Fatalf("%v", got)
	}
	if sent := h.sent(); len(sent) != 1 || sent[0].method != "GET" {
		t.Fatalf("%+v", sent)
	}
}

func TestDryRunDestructiveNeedsNoConfirm(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		method, path string
	}{
		{[]string{"tags", "delete", "--tag-id", "4"}, "DELETE", ws + "/tags/4"},
		{[]string{"triggers", "delete", "--trigger-id", "4"}, "DELETE", ws + "/triggers/4"},
		{[]string{"variables", "delete", "--variable-id", "4"}, "DELETE", ws + "/variables/4"},
		{[]string{"versions", "publish", "--version-id", "9"}, "POST", "accounts/1/containers/2/versions/9:publish"},
	} {
		for _, extra := range [][]string{{"--dry-run"}, {"--dry-run", "--confirm"}} {
			h := newHarness(t).withWorkspace()
			got := h.ok(append(slices.Clone(tc.args), extra...)...)
			url, _ := at(got, "request", "url").(string)
			if at(got, "request", "method") != tc.method || !strings.HasSuffix(url, tc.path) || got["success"] != nil {
				t.Errorf("%v: %v", tc.args, got)
			}
			h.noRequests()
		}
	}
}

func TestDryRunVersionsCreate(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/status", `{"workspaceChange":[{}]}`)
	got := h.ok("versions", "create", "--name", "Release", "--dry-run")
	if at(got, "request", "body", "name") != "Release" || len(h.sent()) != 1 {
		t.Fatalf("%v %+v", got, h.sent())
	}
}

func TestDefaultsPrecedence(t *testing.T) {
	h := newHarness(t).
		on("GET accounts/10/containers/20/workspaces", `{}`).
		on("GET accounts/11/containers/20/workspaces", `{}`).
		on("GET accounts/12/containers/20/workspaces", `{}`)
	path := filepath.Join(h.env["HOME"], ".config", "gtm-cli", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"accountId":"10","containerId":"20"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h.ok("workspaces", "list")
	h.env["GTM_ACCOUNT_ID"] = "11"
	h.ok("workspaces", "list")
	h.ok("workspaces", "list", "--account", "12")
	var paths []string
	for _, r := range h.sent() {
		paths = append(paths, strings.Split(r.path, "/")[1])
	}
	if !slices.Equal(paths, []string{"10", "11", "12"}) {
		t.Fatalf("%v", paths)
	}
}

func TestConfigLocations(t *testing.T) {
	h := newHarness(t).on("GET accounts/5/containers", `{}`)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "gtm-cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gtm-cli", "config.json"), []byte(`{"accountId":"5"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h.env["XDG_CONFIG_HOME"] = dir
	h.ok("containers", "list")
	h.env["XDG_CONFIG_HOME"] = ""
	h.env["GTM_CONFIG"] = filepath.Join(dir, "gtm-cli", "config.json")
	h.ok("containers", "list")
}

func TestBrokenConfig(t *testing.T) {
	h := newHarness(t).on("GET accounts", `{}`)
	h.env["GTM_CONFIG"] = filepath.Join(t.TempDir(), "missing.json")
	h.fails(2, "usage", "containers", "list")
	// Commands that need no IDs must not depend on the config file.
	h.ok("accounts", "list")
	for _, content := range []string{`{"account":"1"}`, `not json`} {
		if err := os.WriteFile(h.env["GTM_CONFIG"], []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		h.fails(2, "usage", "containers", "list")
	}
}

type failingSource struct{}

func (failingSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("oauth2: invalid_grant: Token has been expired or revoked")
}

func TestAuthenticationFailures(t *testing.T) {
	h := newHarness(t).noCredentials()
	if message := h.fails(3, "auth", "accounts", "list"); !strings.Contains(message, "default credentials") {
		t.Fatalf("%q", message)
	}
	h.app.Credentials = func(context.Context) (*auth.Credentials, error) {
		return &auth.Credentials{TokenSource: failingSource{}}, nil
	}
	if message := h.fails(3, "auth", "accounts", "list"); !strings.Contains(message, "invalid_grant") {
		t.Fatalf("%q", message)
	}
	h.fails(3, "auth", "auth", "status")
	h.noRequests()
}

func TestAuthStatus(t *testing.T) {
	scopes := strings.Join(auth.GoogleScopes, " ")
	h := newHarness(t)
	h.on("POST /tokeninfo", `{"scope":"`+scopes+`","expires_in":"100"}`)
	got := h.ok("auth", "status")
	if got["authenticated"] != true || got["email"] != "bot@example.com" || got["source"] != "test" ||
		got["type"] != "service_account" || got["expiresIn"] != float64(100) || len(got["missingScopes"].([]any)) != 0 {
		t.Fatalf("%v", got)
	}

	h.on("POST /tokeninfo", `{"email":"me@example.com","scope":"`+auth.GoogleScopes[0]+`"}`)
	got = h.ok("auth", "status")
	if got["email"] != "me@example.com" || len(got["missingScopes"].([]any)) != len(auth.GoogleScopes)-1 {
		t.Fatalf("%v", got)
	}
}

// A token with no Tag Manager scope cannot do anything here: report the
// identity, and fail so scripts notice.
func TestAuthStatusWithoutTagManagerScopes(t *testing.T) {
	h := newHarness(t).on("POST /tokeninfo", `{"scope":"https://www.googleapis.com/auth/cloud-platform"}`)
	got := h.run("auth", "status")
	if got.exit != 3 || decodeObject(t, got.stdout)["email"] != "bot@example.com" ||
		at(decodeObject(t, got.stderr), "error", "code") != "auth" || !strings.Contains(got.stderr, "--client-id-file") {
		t.Fatalf("%+v", got)
	}
}

func TestTableOutput(t *testing.T) {
	h := newHarness(t).withWorkspace().
		on("GET "+ws+"/tags", `{"tag":[{"tagId":"4","name":"evil\u001b[31m\tname\n\u202e","type":"html"}]}`).
		on("GET "+ws+"/tags/4", `{"tagId":"4","name":"One","type":"html","firingTriggerId":["7"]}`).
		on("GET "+ws+"/status", `{}`)

	got := h.run("tags", "list", "--output", "table")
	lines := strings.Split(strings.TrimRight(got.stdout, "\n"), "\n")
	if got.exit != 0 || len(lines) != 2 || !slices.Equal(strings.Fields(lines[0]), []string{"TAGID", "NAME", "TYPE", "PAUSED"}) {
		t.Fatalf("%+v", got)
	}
	if strings.ContainsAny(got.stdout, "\x1b\t\u202e") {
		t.Fatalf("control characters reached the terminal: %q", got.stdout)
	}

	got = h.run("tags", "get", "--tag-id", "4", "--output=table")
	if got.exit != 0 || !strings.Contains(got.stdout, "firingTriggerId") || !strings.Contains(got.stdout, `["7"]`) {
		t.Fatalf("%+v", got)
	}
	got = h.run("workspaces", "status", "--output", "table")
	if got.exit != 0 || !strings.Contains(got.stdout, "hasChanges") {
		t.Fatalf("%+v", got)
	}
}

func TestJSONKeepsHTMLReadable(t *testing.T) {
	h := newHarness(t).withWorkspace().on("GET "+ws+"/tags/4", `{"tagId":"4","notes":"<b>&</b>"}`)
	if got := h.run("tags", "get", "--tag-id", "4"); !strings.Contains(got.stdout, `"<b>&</b>"`) {
		t.Fatalf("%+v", got)
	}
}
