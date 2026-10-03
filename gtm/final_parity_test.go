package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestVersionLifecycleUsesOfficialRoutes(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/versions/3" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.DeleteVersion(context.Background(), "1", "2", "3"); err != nil {
			t.Fatal(err)
		}
	})

	for _, tc := range []struct {
		name, suffix string
		call         func(*Client) (*VersionLifecycle, error)
	}{
		{"undelete", ":undelete", func(c *Client) (*VersionLifecycle, error) {
			return c.UndeleteVersion(context.Background(), "1", "2", "3")
		}},
		{"set latest", ":set_latest", func(c *Client) (*VersionLifecycle, error) {
			return c.SetLatestVersion(context.Background(), "1", "2", "3")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/versions/3"+tc.suffix {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"containerVersionId":"3","name":"Release","description":"D","fingerprint":"fp"}`)
			})
			got, err := tc.call(client)
			if err != nil || got.VersionID != "3" || got.Description != "D" {
				t.Fatalf("version=%+v err=%v", got, err)
			}
		})
	}

	t.Run("update", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			if calls == 1 {
				if r.Method != http.MethodGet {
					t.Errorf("first method=%s", r.Method)
				}
				fmt.Fprint(w, `{"containerVersionId":"3","name":"Keep","description":"clear","fingerprint":"old"}`)
				return
			}
			if r.Method != http.MethodPut || r.URL.Query().Get("fingerprint") != "old" {
				t.Errorf("unexpected update: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] != "Keep" || body["description"] != "" {
				t.Fatalf("body=%#v err=%v", body, err)
			}
			if _, exists := body["containerVersionId"]; exists {
				t.Errorf("immutable ID sent: %#v", body)
			}
			fmt.Fprint(w, `{"containerVersionId":"3","name":"Keep","fingerprint":"new"}`)
		})
		empty := ""
		got, err := client.UpdateVersion(context.Background(), "1", "2", "3", nil, &empty)
		if err != nil || calls != 2 || got.Fingerprint != "new" {
			t.Fatalf("calls=%d version=%+v err=%v", calls, got, err)
		}
	})
}

func TestEveryWorkspaceResourceRevertUsesFingerprint(t *testing.T) {
	cases := []struct {
		kind, segment, response string
	}{
		{"client", "clients", `{"client":{"clientId":"4","name":"Restored"}}`},
		{"tag", "tags", `{"tag":{"tagId":"4","name":"Restored"}}`},
		{"template", "templates", `{"template":{"templateId":"4","name":"Restored"}}`},
		{"transformation", "transformations", `{"transformation":{"transformationId":"4","name":"Restored"}}`},
		{"trigger", "triggers", `{"trigger":{"triggerId":"4","name":"Restored"}}`},
		{"variable", "variables", `{"variable":{"variableId":"4","name":"Restored"}}`},
		{"zone", "zones", `{"zone":{"zoneId":"4","name":"Restored"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			calls := 0
			client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				expected := "/tagmanager/v2/accounts/1/containers/2/workspaces/3/" + tc.segment + "/4"
				w.Header().Set("Content-Type", "application/json")
				if calls == 1 {
					if r.Method != http.MethodGet || r.URL.Path != expected {
						t.Errorf("unexpected get: %s %s", r.Method, r.URL)
					}
					fmt.Fprint(w, `{"fingerprint":"old"}`)
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != expected+":revert" || r.URL.Query().Get("fingerprint") != "old" {
					t.Errorf("unexpected revert: %s %s", r.Method, r.URL)
				}
				fmt.Fprint(w, tc.response)
			})
			got, err := client.RevertWorkspaceEntity(context.Background(), "1", "2", "3", tc.kind, "4")
			if err != nil || calls != 2 || !got.ExistsAfterRevert || got.Resource == nil {
				t.Fatalf("calls=%d result=%+v err=%v", calls, got, err)
			}
		})
	}

	t.Run("built-in variable", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/built_in_variables:revert" || r.URL.Query().Get("type") != "pageUrl" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"enabled":true}`)
		})
		got, err := client.RevertWorkspaceEntity(context.Background(), "1", "2", "3", "builtInVariable", "pageUrl")
		if err != nil || !got.ExistsAfterRevert {
			t.Fatalf("result=%+v err=%v", got, err)
		}
	})
}

func TestFinalParityConfirmationGuardsRunBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerDeleteVersion(server)
	registerUndeleteVersion(server)
	registerSetLatestVersion(server)
	registerRevertWorkspaceEntity(server)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	cases := []struct {
		name string
		args map[string]any
	}{
		{"delete_version", map[string]any{"accountId": "1", "containerId": "2", "versionId": "3", "confirm": false}},
		{"undelete_version", map[string]any{"accountId": "1", "containerId": "2", "versionId": "3", "confirm": false}},
		{"set_latest_version", map[string]any{"accountId": "1", "containerId": "2", "versionId": "3", "confirm": false}},
		{"revert_workspace_entity", map[string]any{"accountId": "1", "containerId": "2", "workspaceId": "3", "resourceType": "tag", "resourceId": "4", "confirm": false}},
	}
	for _, tc := range cases {
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		encoded, _ := json.Marshal(result)
		if strings.Contains(string(encoded), "not authenticated") {
			t.Fatalf("%s reached authentication before confirmation: %s", tc.name, encoded)
		}
	}
}
