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

func TestDestinationsUseOfficialRoutes(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/destinations" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"destination":[{"destinationId":"G-ABC","destinationLinkId":"4","name":"Web","fingerprint":"fp"}]}`)
		})
		got, err := client.ListDestinations(context.Background(), "1", "2")
		if err != nil || len(got) != 1 || got[0].DestinationID != "G-ABC" || got[0].DestinationLinkID != "4" {
			t.Fatalf("destinations=%+v err=%v", got, err)
		}
	})

	t.Run("get", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/destinations/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"destinationId":"G-ABC","destinationLinkId":"4","name":"Web"}`)
		})
		got, err := client.GetDestination(context.Background(), "1", "2", "4")
		if err != nil || got.Name != "Web" {
			t.Fatalf("destination=%+v err=%v", got, err)
		}
	})

	t.Run("link", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/destinations:link" || r.URL.Query().Get("destinationId") != "G-ABC" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			if r.URL.Query().Has("allowUserPermissionFeatureUpdate") {
				t.Errorf("permission feature update was unexpectedly enabled: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"destinationId":"G-ABC","destinationLinkId":"4"}`)
		})
		got, err := client.LinkDestination(context.Background(), "1", "2", "G-ABC")
		if err != nil || got.DestinationLinkID != "4" {
			t.Fatalf("destination=%+v err=%v", got, err)
		}
	})
}

func TestContainerAdminOperationsUseSafeOfficialParameters(t *testing.T) {
	t.Run("combine", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2:combine" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			if r.URL.Query().Get("containerId") != "3" || r.URL.Query().Get("settingSource") != "current" || r.URL.Query().Has("allowUserPermissionFeatureUpdate") {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"containerId":"2","name":"Combined","publicId":"GTM-ABC"}`)
		})
		got, err := client.CombineContainers(context.Background(), "1", "2", "3", "current")
		if err != nil || got.ContainerID != "2" || got.Name != "Combined" {
			t.Fatalf("container=%+v err=%v", got, err)
		}
	})

	t.Run("move tag ID", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2:move_tag_id" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			query := r.URL.Query()
			if query.Get("tagId") != "G-ABC" || query.Get("tagName") != "New container" || query.Get("copySettings") != "true" || query.Get("copyTermsOfService") != "true" || query.Get("copyUsers") != "false" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			if query.Has("allowUserPermissionFeatureUpdate") {
				t.Errorf("permission feature update was unexpectedly enabled: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"containerId":"3","name":"New container","publicId":"GTM-NEW"}`)
		})
		got, err := client.MoveTagID(context.Background(), "1", "2", "G-ABC", "New container", true)
		if err != nil || got.ContainerID != "3" || got.PublicID != "GTM-NEW" {
			t.Fatalf("container=%+v err=%v", got, err)
		}
	})
}

func TestGoogleTagConfigCRUDUsesOfficialRoutes(t *testing.T) {
	t.Run("list paginates", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/gtag_config" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("pageToken") == "" {
				fmt.Fprint(w, `{"gtagConfig":[{"gtagConfigId":"4","type":"googleTag","parameter":[{"type":"template","key":"tagId","value":"G-ABC"}]}],"nextPageToken":"next"}`)
			} else {
				fmt.Fprint(w, `{"gtagConfig":[{"gtagConfigId":"5","type":"googleTag"}]}`)
			}
		})
		got, err := client.ListGoogleTagConfigs(context.Background(), "1", "2", "3")
		if err != nil || calls != 2 || len(got) != 2 || got[0].ConfigID != "4" || got[0].Parameter == nil {
			t.Fatalf("calls=%d configs=%+v err=%v", calls, got, err)
		}
	})

	t.Run("get", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/gtag_config/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"gtagConfigId":"4","type":"googleTag","fingerprint":"fp"}`)
		})
		got, err := client.GetGoogleTagConfig(context.Background(), "1", "2", "3", "4")
		if err != nil || got.Fingerprint != "fp" {
			t.Fatalf("config=%+v err=%v", got, err)
		}
	})

	t.Run("create", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/gtag_config" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["type"] != "googleTag" || body["parameter"] == nil {
				t.Fatalf("unexpected body: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"gtagConfigId":"4","type":"googleTag"}`)
		})
		got, err := client.CreateGoogleTagConfig(context.Background(), "1", "2", "3", GoogleTagConfigCreateConfig{
			Type: "googleTag", Parameter: []Parameter{{Type: "template", Key: "tagId", Value: "G-ABC"}},
		})
		if err != nil || got.ConfigID != "4" {
			t.Fatalf("config=%+v err=%v", got, err)
		}
	})

	t.Run("update preserves and clears", func(t *testing.T) {
		calls := 0
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/gtag_config/4" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			if calls == 1 {
				fmt.Fprint(w, `{"gtagConfigId":"4","type":"googleTag","parameter":[{"type":"template","key":"tagId","value":"G-ABC"}],"fingerprint":"old"}`)
				return
			}
			if r.Method != http.MethodPut || r.URL.Query().Get("fingerprint") != "old" {
				t.Errorf("unexpected update: %s %s", r.Method, r.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["type"] != "googleTag" {
				t.Errorf("type not preserved: %#v", body)
			}
			if parameters, ok := body["parameter"].([]any); !ok || len(parameters) != 0 {
				t.Errorf("parameters not cleared: %#v", body)
			}
			if _, present := body["gtagConfigId"]; present {
				t.Errorf("immutable ID sent: %#v", body)
			}
			fmt.Fprint(w, `{"gtagConfigId":"4","type":"googleTag","fingerprint":"new"}`)
		})
		empty := []Parameter{}
		got, err := client.UpdateGoogleTagConfig(context.Background(), "1", "2", "3", "4", GoogleTagConfigUpdateConfig{Parameter: &empty})
		if err != nil || calls != 2 || got.Fingerprint != "new" {
			t.Fatalf("calls=%d config=%+v err=%v", calls, got, err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/gtag_config/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.DeleteGoogleTagConfig(context.Background(), "1", "2", "3", "4"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestContainerAdminConfirmationGuardsRunBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerLinkDestination(server)
	registerCombineContainers(server)
	registerMoveTagID(server)
	registerDeleteGoogleTagConfig(server)
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
		{"link_destination", map[string]any{"accountId": "1", "containerId": "2", "destinationId": "G-ABC", "confirm": false}},
		{"combine_containers", map[string]any{"accountId": "1", "containerId": "2", "sourceContainerId": "3", "settingSource": "current", "confirm": false}},
		{"move_tag_id", map[string]any{"accountId": "1", "containerId": "2", "tagId": "G-ABC", "tagName": "new", "acceptTerms": true, "confirm": false}},
		{"delete_google_tag_config", map[string]any{"accountId": "1", "containerId": "2", "workspaceId": "3", "gtagConfigId": "4", "confirm": false}},
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
