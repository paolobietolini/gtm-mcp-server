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

func TestFolderReadsPaginateAndGetExactPath(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("pageToken") == "" {
				fmt.Fprint(w, `{"folder":[{"folderId":"4","name":"A","fingerprint":"fp"}],"nextPageToken":"next"}`)
			} else {
				fmt.Fprint(w, `{"folder":[{"folderId":"5","name":"B"}]}`)
			}
		})
		got, err := client.ListFolders(context.Background(), "1", "2", "3")
		if err != nil || calls != 2 || len(got) != 2 || got[0].Fingerprint != "fp" {
			t.Fatalf("calls=%d folders=%+v err=%v", calls, got, err)
		}
	})

	t.Run("entities", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders/4:entities" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("pageToken") == "" {
				fmt.Fprint(w, `{"tag":[{"name":"Tag A"}],"nextPageToken":"next"}`)
			} else {
				fmt.Fprint(w, `{"trigger":[{"name":"Trigger B"}],"variable":[{"name":"Variable C"}]}`)
			}
		})
		got, err := client.GetFolderEntities(context.Background(), "1", "2", "3", "4")
		if err != nil || calls != 2 || len(got.Tags) != 1 || len(got.Triggers) != 1 || len(got.Variables) != 1 {
			t.Fatalf("calls=%d entities=%+v err=%v", calls, got, err)
		}
	})

	t.Run("get", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"folderId":"4","name":"A","notes":"N","fingerprint":"fp"}`)
		})
		got, err := client.GetFolder(context.Background(), "1", "2", "3", "4")
		if err != nil || got.Name != "A" || got.Notes != "N" {
			t.Fatalf("folder=%+v err=%v", got, err)
		}
	})
}

func TestFolderMutationsUseFingerprintAndOfficialRoutes(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] != "Marketing" || body["notes"] != "Owned" {
				t.Fatalf("body=%#v err=%v", body, err)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"folderId":"4","name":"Marketing"}`)
		})
		got, err := client.CreateFolder(context.Background(), "1", "2", "3", "Marketing", "Owned")
		if err != nil || got.FolderID != "4" {
			t.Fatalf("folder=%+v err=%v", got, err)
		}
	})

	t.Run("update", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			if calls == 1 {
				fmt.Fprint(w, `{"folderId":"4","name":"Keep","notes":"clear","fingerprint":"old"}`)
				return
			}
			if r.Method != http.MethodPut || r.URL.Query().Get("fingerprint") != "old" {
				t.Errorf("unexpected update: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] != "Keep" || body["notes"] != "" {
				t.Fatalf("body=%#v err=%v", body, err)
			}
			if _, exists := body["folderId"]; exists {
				t.Errorf("immutable ID sent: %#v", body)
			}
			fmt.Fprint(w, `{"folderId":"4","name":"Keep","fingerprint":"new"}`)
		})
		empty := ""
		got, err := client.UpdateFolder(context.Background(), "1", "2", "3", "4", nil, &empty)
		if err != nil || calls != 2 || got.Fingerprint != "new" {
			t.Fatalf("calls=%d folder=%+v err=%v", calls, got, err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.DeleteFolder(context.Background(), "1", "2", "3", "4"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("move entities", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders/4:move_entities_to_folder" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			if len(r.URL.Query()["tagId"]) != 2 || r.URL.Query().Get("triggerId") != "7" || r.URL.Query().Get("variableId") != "8" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.MoveEntitiesToFolder(context.Background(), "1", "2", "3", "4", []string{"5", "6"}, []string{"7"}, []string{"8"}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("revert", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			if calls == 1 {
				fmt.Fprint(w, `{"folderId":"4","fingerprint":"old"}`)
				return
			}
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/folders/4:revert" || r.URL.Query().Get("fingerprint") != "old" {
				t.Errorf("unexpected revert: %s %s", r.Method, r.URL)
			}
			fmt.Fprint(w, `{"folder":{"folderId":"4","name":"Restored"}}`)
		})
		got, err := client.RevertFolder(context.Background(), "1", "2", "3", "4")
		if err != nil || calls != 2 || got.Name != "Restored" {
			t.Fatalf("calls=%d folder=%+v err=%v", calls, got, err)
		}
	})
}

func TestWorkspaceOperationsUseOfficialPayloadsAndRoutes(t *testing.T) {
	t.Run("bulk update", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/bulk_update" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["changes"] == nil {
				t.Fatalf("body=%#v err=%v", body, err)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"changes":[{"changeStatus":"added","folder":{"folderId":"new_1","name":"New"}}]}`)
		})
		changes, err := client.BulkUpdateWorkspace(context.Background(), "1", "2", "3", `{"changes":[{"changeStatus":"added","folder":{"folderId":"new_1","name":"New"}}]}`)
		if err != nil || changes == nil {
			t.Fatalf("changes=%+v err=%v", changes, err)
		}
	})

	t.Run("resolve conflict", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3:resolve_conflict" || r.URL.Query().Get("fingerprint") != "fp" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.ResolveWorkspaceConflict(context.Background(), "1", "2", "3", "fp", `{"changeStatus":"updated","folder":{"folderId":"4","name":"Resolved"}}`); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sync", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3:sync" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"syncStatus":{"mergeConflict":true,"syncError":false},"mergeConflict":[{"entityInWorkspace":{"changeStatus":"updated"}}]}`)
		})
		got, err := client.SyncWorkspace(context.Background(), "1", "2", "3")
		if err != nil || !got.MergeConflict || got.SyncError || got.Conflicts == nil {
			t.Fatalf("result=%+v err=%v", got, err)
		}
	})
}

func TestWorkspaceFolderConfirmationGuardsRunBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerDeleteFolder(server)
	registerMoveEntitiesToFolder(server)
	registerRevertFolder(server)
	registerBulkUpdateWorkspace(server)
	registerResolveWorkspaceConflict(server)
	registerSyncWorkspace(server)
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

	workspace := map[string]any{"accountId": "1", "containerId": "2", "workspaceId": "3", "confirm": false}
	cases := []struct {
		name string
		args map[string]any
	}{
		{"delete_folder", mergeArgs(workspace, map[string]any{"folderId": "4"})},
		{"move_entities_to_folder", mergeArgs(workspace, map[string]any{"folderId": "4", "tagIds": []string{"5"}})},
		{"revert_folder", mergeArgs(workspace, map[string]any{"folderId": "4"})},
		{"bulk_update_workspace", mergeArgs(workspace, map[string]any{"changesJson": `{"changes":[]}`})},
		{"resolve_workspace_conflict", mergeArgs(workspace, map[string]any{"fingerprint": "fp", "entityJson": `{}`})},
		{"sync_workspace", workspace},
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

func mergeArgs(base, extra map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}
