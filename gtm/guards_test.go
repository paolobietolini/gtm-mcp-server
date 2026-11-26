package gtm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestCheckConfirm(t *testing.T) {
	for op, want := range map[Operation]string{
		OpDeleteTag:      "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		OpDeleteTrigger:  "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		OpDeleteVariable: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		OpPublishVersion: "Publishing requires confirm: true. WARNING: This will make the version live on your website.",
	} {
		err := CheckConfirm(op, false)
		if !errors.Is(err, ErrConfirmationRequired) || err.Error() != want {
			t.Errorf("%s: %v", op, err)
		}
		if err := CheckConfirm(op, true); err != nil {
			t.Errorf("%s confirmed: %v", op, err)
		}
	}
	// An operation without its own message must still be guarded.
	if err := CheckConfirm("unknown_operation", false); !errors.Is(err, ErrConfirmationRequired) {
		t.Errorf("unknown operation allowed: %v", err)
	}
}

func TestCreateVersionFromWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name, status, blocked string
	}{
		{"no changes", `{}`, "No changes in workspace to create version from"},
		{"conflicts", `{"workspaceChange":[{}],"mergeConflict":[{},{}]}`, "Workspace has 2 conflicts that must be resolved before creating a version"},
		{"ready", `{"workspaceChange":[{}]}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created := false
			client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, tc.status)
					return
				}
				created = true
				fmt.Fprint(w, `{"containerVersion":{"containerVersionId":"9"}}`)
			})
			version, err := client.CreateVersionFromWorkspace(context.Background(), "1", "2", "3", &VersionInput{Name: "v"})
			var blocked *VersionBlockedError
			if tc.blocked == "" {
				if err != nil || !created || version.VersionID != "9" {
					t.Fatalf("version=%+v created=%v err=%v", version, created, err)
				}
			} else if !errors.As(err, &blocked) || blocked.Error() != tc.blocked || created {
				t.Fatalf("created=%v err=%v", created, err)
			}
		})
	}
}

func TestNormalizeTriggerUpdate(t *testing.T) {
	a := Condition{Type: "equals"}
	b := Condition{Type: "contains"}
	input := &TriggerInput{Type: "linkClick", Filter: []Condition{a}, AutoEventFilter: []Condition{b}}
	warning := NormalizeTriggerUpdate(input)
	if len(input.Filter) != 2 || input.AutoEventFilter != nil || warning == "" {
		t.Fatalf("input=%+v warning=%q", input, warning)
	}
	input = &TriggerInput{Type: "timer", AutoEventFilter: []Condition{b}}
	if warning := NormalizeTriggerUpdate(input); warning != "" || len(input.AutoEventFilter) != 1 {
		t.Fatalf("timer was rewritten: %+v %q", input, warning)
	}
}

func TestTriggerCreateMessage(t *testing.T) {
	if got := TriggerCreateMessage(&TriggerInput{Type: "pageview"}); got != "Trigger created successfully" {
		t.Fatalf("got %q", got)
	}
	want := "Trigger created successfully. Note: autoEventFilter was automatically remapped to filter for click triggers (GTM API requirement)."
	if got := TriggerCreateMessage(&TriggerInput{Type: "click", AutoEventFilter: []Condition{{}}}); got != want {
		t.Fatalf("got %q", got)
	}
}
