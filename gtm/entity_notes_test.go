package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

func TestEntityReadMappingsIncludeNotesAndParentFolder(t *testing.T) {
	tag := toTag(&tagmanager.Tag{
		TagId:          "tag-1",
		Notes:          "tag notes",
		ParentFolderId: "folder-1",
	})
	if tag.Notes != "tag notes" || tag.ParentFolderID != "folder-1" {
		t.Fatalf("tag mapping lost notes or folder: %+v", tag)
	}

	variable := toVariable(&tagmanager.Variable{
		VariableId:     "variable-1",
		Notes:          "variable notes",
		ParentFolderId: "folder-2",
	})
	if variable.Notes != "variable notes" || variable.ParentFolderID != "folder-2" {
		t.Fatalf("variable mapping lost notes or folder: %+v", variable)
	}
}

func captureEntityUpdate(t *testing.T, currentJSON string, update func(*Client) error) map[string]any {
	t.Helper()
	var body map[string]any
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = fmt.Fprint(w, currentJSON)
		case http.MethodPut:
			if r.URL.Query().Get("fingerprint") != "original" {
				t.Errorf("missing fingerprint in update URL: %s", r.URL)
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			_, _ = fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})

	if err := update(client); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("got %d API calls, want GET then PUT", calls)
	}
	return body
}

func TestUpdateTagPreservesOmittedNotesAndAllowsClear(t *testing.T) {
	for _, clear := range []bool{false, true} {
		name := "preserve"
		if clear {
			name = "clear"
		}
		t.Run(name, func(t *testing.T) {
			body := captureEntityUpdate(t,
				`{"tagId":"4","name":"Old","type":"html","notes":"keep","parentFolderId":"folder","fingerprint":"original"}`,
				func(client *Client) error {
					_, err := client.UpdateTag(context.Background(), "accounts/1/containers/2/workspaces/3/tags/4", &TagInput{
						Name: "New", Type: "html", HasNotes: clear,
					})
					return err
				})

			wantNotes := "keep"
			if clear {
				wantNotes = ""
			}
			if notes, present := body["notes"]; !present || notes != wantNotes {
				t.Errorf("notes=%#v present=%v, want %q: %#v", notes, present, wantNotes, body)
			}
			if body["parentFolderId"] != "folder" {
				t.Errorf("parent folder was not preserved: %#v", body)
			}
		})
	}
}

func TestUpdateTriggerPreservesOmittedNotesAndParentFolder(t *testing.T) {
	for _, clear := range []bool{false, true} {
		name := "preserve"
		if clear {
			name = "clear"
		}
		t.Run(name, func(t *testing.T) {
			body := captureEntityUpdate(t,
				`{"triggerId":"4","name":"Old","type":"pageview","notes":"keep","parentFolderId":"folder","fingerprint":"original"}`,
				func(client *Client) error {
					_, err := client.UpdateTrigger(context.Background(), "accounts/1/containers/2/workspaces/3/triggers/4", &TriggerInput{
						Name: "New", Type: "pageview", HasNotes: clear,
					})
					return err
				})

			wantNotes := "keep"
			if clear {
				wantNotes = ""
			}
			if notes, present := body["notes"]; !present || notes != wantNotes {
				t.Errorf("notes=%#v present=%v, want %q: %#v", notes, present, wantNotes, body)
			}
			if body["parentFolderId"] != "folder" {
				t.Errorf("parent folder was not preserved: %#v", body)
			}
		})
	}
}

func TestUpdateVariablePreservesOmittedFieldsAndAllowsClear(t *testing.T) {
	for _, clear := range []bool{false, true} {
		name := "preserve"
		if clear {
			name = "clear"
		}
		t.Run(name, func(t *testing.T) {
			body := captureEntityUpdate(t,
				`{"variableId":"4","name":"Old","type":"c","notes":"keep","parentFolderId":"folder","parameter":[{"key":"value","type":"template","value":"old"}],"fingerprint":"original"}`,
				func(client *Client) error {
					_, err := client.UpdateVariable(context.Background(), "accounts/1/containers/2/workspaces/3/variables/4", &VariableInput{
						Name: "New", Type: "c", HasNotes: clear, HasParameter: clear,
					})
					return err
				})

			wantNotes := "keep"
			if clear {
				wantNotes = ""
			}
			if notes, present := body["notes"]; !present || notes != wantNotes {
				t.Errorf("notes=%#v present=%v, want %q: %#v", notes, present, wantNotes, body)
			}
			if body["parentFolderId"] != "folder" {
				t.Errorf("parent folder was not preserved: %#v", body)
			}
			parameters, present := body["parameter"].([]any)
			if !present {
				t.Fatalf("parameter missing or invalid: %#v", body)
			}
			wantLength := 1
			if clear {
				wantLength = 0
			}
			if len(parameters) != wantLength {
				t.Errorf("parameter length=%d, want %d: %#v", len(parameters), wantLength, body)
			}
		})
	}
}
