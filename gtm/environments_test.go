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

func TestListEnvironmentsPaginatesAndMapsFields(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"environment":[{"accountId":"1","containerId":"2","environmentId":"3","name":"QA","type":"user","url":"https://example.com","enableDebug":true,"authorizationCode":"code","fingerprint":"fp"}],"nextPageToken":"next"}`)
		case "next":
			fmt.Fprint(w, `{"environment":[{"environmentId":"4","name":"Live","type":"live","containerVersionId":"9"}]}`)
		default:
			t.Fatalf("unexpected page token %q", r.URL.Query().Get("pageToken"))
		}
	})

	environments, err := client.ListEnvironments(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(environments) != 2 || environments[0].AuthorizationCode != "code" || environments[1].ContainerVersionID != "9" {
		t.Fatalf("calls=%d environments=%+v", calls, environments)
	}
}

func TestGetEnvironmentUsesExactPath(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"environmentId":"3","name":"QA","type":"user","workspaceId":"7","fingerprint":"fp"}`)
	})
	environment, err := client.GetEnvironment(context.Background(), "1", "2", "3")
	if err != nil {
		t.Fatal(err)
	}
	if environment.EnvironmentID != "3" || environment.Name != "QA" || environment.WorkspaceID != "7" {
		t.Fatalf("unexpected environment: %+v", environment)
	}
}

func TestCreateEnvironmentSendsUserConfiguration(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "QA" || body["type"] != "user" || body["enableDebug"] != false || body["url"] != "https://example.com" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"environmentId":"3","name":"QA","type":"user","fingerprint":"fp"}`)
	})

	environment, err := client.CreateEnvironment(context.Background(), "1", "2", EnvironmentCreateConfig{
		Name: "QA", Description: "Quality assurance", URL: "https://example.com", EnableDebug: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if environment.EnvironmentID != "3" || environment.Fingerprint != "fp" {
		t.Fatalf("unexpected environment: %+v", environment)
	}
}

func TestUpdateEnvironmentPreservesOmittedAndClearsFields(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments/3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			if r.Method != http.MethodGet {
				t.Errorf("first method=%s", r.Method)
			}
			fmt.Fprint(w, `{"environmentId":"3","name":"Keep","description":"clear","url":"https://old.example","enableDebug":true,"containerVersionId":"8","workspaceId":"7","type":"user","fingerprint":"old"}`)
			return
		}
		if r.Method != http.MethodPut || r.URL.Query().Get("fingerprint") != "old" {
			t.Errorf("update request: %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "Keep" || body["description"] != "" || body["url"] != "" || body["enableDebug"] != false || body["workspaceId"] != "" {
			t.Fatalf("fields not preserved or cleared: %#v", body)
		}
		if body["containerVersionId"] != "8" {
			t.Errorf("omitted containerVersionId was not preserved: %#v", body)
		}
		if _, present := body["environmentId"]; present {
			t.Errorf("update sent immutable response fields: %#v", body)
		}
		fmt.Fprint(w, `{"environmentId":"3","name":"Keep","fingerprint":"new"}`)
	})

	empty := ""
	disableDebug := false
	environment, err := client.UpdateEnvironment(context.Background(), "1", "2", "3", EnvironmentUpdateConfig{
		Description: &empty, URL: &empty, EnableDebug: &disableDebug, WorkspaceID: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || environment.Fingerprint != "new" {
		t.Fatalf("calls=%d environment=%+v", calls, environment)
	}
}

func TestReauthorizeAndDeleteEnvironmentUseExactPaths(t *testing.T) {
	t.Run("reauthorize", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments/3:reauthorize" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"environmentId":"3","authorizationCode":"new-code"}`)
		})
		environment, err := client.ReauthorizeEnvironment(context.Background(), "1", "2", "3")
		if err != nil {
			t.Fatal(err)
		}
		if environment.AuthorizationCode != "new-code" {
			t.Fatalf("unexpected environment: %+v", environment)
		}
	})

	t.Run("delete", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/environments/3" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.DeleteEnvironment(context.Background(), "1", "2", "3"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"code":404,"message":"missing environment"}}`)
		})
		if _, err := client.GetEnvironment(context.Background(), "1", "2", "3"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("error=%v, want ErrNotFound", err)
		}
	})
}

func TestEnvironmentConfirmationGuardsRunBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerReauthorizeEnvironment(server)
	registerDeleteEnvironment(server)
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

	for _, name := range []string{"reauthorize_environment", "delete_environment"} {
		result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{
			"accountId": "1", "containerId": "2", "environmentId": "3", "confirm": false,
		}})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			t.Fatalf("%s confirmation guard returned protocol error: %+v", name, result)
		}
	}
}
