package gtm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListWorkspacesPaginationAndFields(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"workspace":[{"accountId":"1","containerId":"2","workspaceId":"3","name":"First","fingerprint":"fp3","tagManagerUrl":"https://tagmanager.example/3"}],"nextPageToken":"next"}`)
		case "next":
			fmt.Fprint(w, `{"workspace":[{"workspaceId":"4","name":"Second"}]}`)
		default:
			t.Errorf("unexpected page token: %q", r.URL.Query().Get("pageToken"))
		}
	})

	got, err := client.ListWorkspaces(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got) != 2 {
		t.Fatalf("calls=%d workspaces=%+v", calls, got)
	}
	if got[0].AccountID != "1" || got[0].Fingerprint != "fp3" || got[0].TagManagerURL == "" {
		t.Fatalf("workspace fields were lost: %+v", got[0])
	}
}

func TestGetWorkspace(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"accountId":"1","containerId":"2","workspaceId":"3","name":"Main","description":"Current","fingerprint":"fp3","path":"accounts/1/containers/2/workspaces/3"}`)
	})

	got, err := client.GetWorkspace(context.Background(), "1", "2", "3")
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != "3" || got.Description != "Current" || got.Fingerprint != "fp3" {
		t.Fatalf("unexpected workspace: %+v", got)
	}
}

func TestUpdateWorkspacePreservesNameAndClearsDescription(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			if r.Method != http.MethodGet {
				t.Errorf("first method=%s", r.Method)
			}
			fmt.Fprint(w, `{"workspaceId":"3","name":"Keep me","description":"Clear me","fingerprint":"current-fp"}`)
		case 2:
			if r.Method != http.MethodPut {
				t.Errorf("second method=%s", r.Method)
			}
			if got := r.URL.Query().Get("fingerprint"); got != "current-fp" {
				t.Errorf("fingerprint=%q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["name"] != "Keep me" {
				t.Errorf("name not preserved: %v", body)
			}
			if value, present := body["description"]; !present || value != "" {
				t.Errorf("description not explicitly cleared: %v", body)
			}
			fmt.Fprint(w, `{"workspaceId":"3","name":"Keep me","fingerprint":"new-fp"}`)
		default:
			t.Fatalf("unexpected call %d", calls)
		}
	})

	empty := ""
	got, err := client.UpdateWorkspace(context.Background(), "1", "2", "3", nil, &empty)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || got.Name != "Keep me" || got.Description != "" || got.Fingerprint != "new-fp" {
		t.Fatalf("calls=%d workspace=%+v", calls, got)
	}
}

func TestUpdateWorkspaceRequiresAChange(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("validation should prevent request: %s", r.URL)
	})
	if _, err := client.UpdateWorkspace(context.Background(), "1", "2", "3", nil, nil); err == nil {
		t.Fatal("UpdateWorkspace succeeded without a change")
	}
}

func TestDeleteWorkspace(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.DeleteWorkspace(context.Background(), "1", "2", "3"); err != nil {
		t.Fatal(err)
	}
}

func TestQuickPreviewWorkspace(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3:quick_preview" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compilerError":true,"syncStatus":{"mergeConflict":true,"syncError":false},"containerVersion":{"name":"Quick Preview","tag":[{"tagId":"10","name":"Preview tag"}]}}`)
	})

	got, err := client.QuickPreviewWorkspace(context.Background(), "1", "2", "3")
	if err != nil {
		t.Fatal(err)
	}
	if !got.CompilerError || got.SyncStatus == nil || !got.SyncStatus.MergeConflict {
		t.Fatalf("unexpected preview status: %+v", got)
	}
	if got.Version == nil || got.Version.Name != "Quick Preview" || got.Version.Tags == nil {
		t.Fatalf("preview version was lost: %+v", got.Version)
	}
}

func TestWorkspaceMetadataErrors(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":404,"message":"missing workspace"}}`)
	})
	name := "New"
	checks := []struct {
		name string
		call func() error
	}{
		{name: "get", call: func() error { _, err := client.GetWorkspace(context.Background(), "1", "2", "3"); return err }},
		{name: "update", call: func() error {
			_, err := client.UpdateWorkspace(context.Background(), "1", "2", "3", &name, nil)
			return err
		}},
		{name: "delete", call: func() error { return client.DeleteWorkspace(context.Background(), "1", "2", "3") }},
		{name: "preview", call: func() error { _, err := client.QuickPreviewWorkspace(context.Background(), "1", "2", "3"); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ErrNotFound) {
				t.Fatalf("error=%v, want ErrNotFound", err)
			}
		})
	}
}

func TestDeleteWorkspaceRequiresConfirmationBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerDeleteWorkspace(server)
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

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "delete_workspace",
		Arguments: map[string]any{
			"accountId": "1", "containerId": "2", "workspaceId": "3", "confirm": false,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("confirmation guard returned protocol error: %+v", result)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output DeleteWorkspaceOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	if output.Success || output.Message == "" {
		t.Fatalf("unexpected output: %+v", output)
	}
}
