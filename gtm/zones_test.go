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

func TestListZonesPaginatesAndMapsConfiguration(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/zones" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"zone":[{"accountId":"1","containerId":"2","workspaceId":"3","zoneId":"4","name":"First","fingerprint":"fp4","boundary":{"customEvaluationTriggerId":["10"]}}],"nextPageToken":"next"}`)
		case "next":
			fmt.Fprint(w, `{"zone":[{"zoneId":"5","name":"Second","childContainer":[{"publicId":"GTM-ABC"}]}]}`)
		default:
			t.Fatalf("unexpected page token %q", r.URL.Query().Get("pageToken"))
		}
	})

	zones, err := client.ListZones(context.Background(), "1", "2", "3")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(zones) != 2 || zones[0].Fingerprint != "fp4" || zones[0].Boundary == nil || zones[1].ChildContainers == nil {
		t.Fatalf("calls=%d zones=%+v", calls, zones)
	}
}

func TestGetZoneUsesExactPath(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/zones/4" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"zoneId":"4","name":"A zone","notes":"notes","fingerprint":"fp"}`)
	})
	zone, err := client.GetZone(context.Background(), "1", "2", "3", "4")
	if err != nil {
		t.Fatal(err)
	}
	if zone.ZoneID != "4" || zone.Name != "A zone" || zone.Notes != "notes" {
		t.Fatalf("unexpected zone: %+v", zone)
	}
}

func TestCreateZoneSendsConfiguration(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/zones" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		boundary, _ := body["boundary"].(map[string]any)
		children, _ := body["childContainer"].([]any)
		restriction, _ := body["typeRestriction"].(map[string]any)
		if body["name"] != "Marketing zone" || len(children) != 1 || boundary["condition"] == nil || restriction["enable"] != false {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"zoneId":"4","name":"Marketing zone","fingerprint":"fp"}`)
	})

	zone, err := client.CreateZone(context.Background(), "1", "2", "3", ZoneCreateConfig{
		Name: "Marketing zone",
		BoundaryConditions: []Condition{{Type: "equals", Parameter: []Parameter{
			{Type: "template", Key: "arg0", Value: "{{Page Hostname}}"},
			{Type: "template", Key: "arg1", Value: "example.com"},
		}}},
		ChildContainers: []ZoneChildContainerInput{{PublicID: "GTM-ABC", Nickname: "Child"}},
		TypeRestriction: &ZoneTypeRestrictionInput{Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if zone.ZoneID != "4" || zone.Fingerprint != "fp" {
		t.Fatalf("unexpected zone: %+v", zone)
	}
}

func TestUpdateZonePreservesOmittedAndClearsCollections(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/zones/4" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			if r.Method != http.MethodGet {
				t.Errorf("first method=%s", r.Method)
			}
			fmt.Fprint(w, `{"zoneId":"4","name":"Keep","notes":"clear","fingerprint":"old","boundary":{"condition":[{"type":"equals"}],"customEvaluationTriggerId":["10"]},"childContainer":[{"publicId":"GTM-OLD"}]}`)
			return
		}
		if r.Method != http.MethodPut || r.URL.Query().Get("fingerprint") != "old" {
			t.Errorf("update request: %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		boundary, _ := body["boundary"].(map[string]any)
		if body["name"] != "Keep" || body["notes"] != "" {
			t.Errorf("scalar fields not preserved/cleared: %#v", body)
		}
		if _, present := body["zoneId"]; present {
			t.Errorf("update sent immutable response fields: %#v", body)
		}
		if conditions, ok := boundary["condition"].([]any); !ok || len(conditions) != 0 {
			t.Errorf("conditions not explicitly cleared: %#v", boundary)
		}
		if children, ok := body["childContainer"].([]any); !ok || len(children) != 0 {
			t.Errorf("children not explicitly cleared: %#v", body)
		}
		fmt.Fprint(w, `{"zoneId":"4","name":"Keep","fingerprint":"new"}`)
	})

	emptyNotes := ""
	emptyConditions := []Condition{}
	emptyChildren := []ZoneChildContainerInput{}
	zone, err := client.UpdateZone(context.Background(), "1", "2", "3", "4", ZoneUpdateConfig{
		Notes: &emptyNotes, BoundaryConditions: &emptyConditions, ChildContainers: &emptyChildren,
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || zone.Name != "Keep" || zone.Fingerprint != "new" {
		t.Fatalf("calls=%d zone=%+v", calls, zone)
	}
}

func TestDeleteZoneAndErrorMapping(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/workspaces/3/zones/4" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		if err := client.DeleteZone(context.Background(), "1", "2", "3", "4"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"code":404,"message":"missing zone"}}`)
		})
		if _, err := client.GetZone(context.Background(), "1", "2", "3", "4"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("error=%v, want ErrNotFound", err)
		}
	})
}

func TestDeleteZoneRequiresConfirmationBeforeAuth(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerDeleteZone(server)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "delete_zone", Arguments: map[string]any{
		"accountId": "1", "containerId": "2", "workspaceId": "3", "zoneId": "4", "confirm": false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("confirmation guard returned protocol error: %+v", result)
	}
	data, _ := json.Marshal(result.StructuredContent)
	var output DeleteZoneOutput
	if err := json.Unmarshal(data, &output); err != nil || output.Success || output.Message == "" {
		t.Fatalf("unexpected output: %+v, err=%v", output, err)
	}
}
