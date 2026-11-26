package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// The GTM API drops autoEventFilter on click and form triggers, so every
// condition must reach it in filter, whichever input field carried it.
func TestCreateTriggerKeepsAllClickConditions(t *testing.T) {
	condition := func(value string) Condition {
		return Condition{Type: "contains", Parameter: []Parameter{{Type: "template", Key: "arg1", Value: value}}}
	}
	for _, triggerType := range []string{"click", "linkClick", "formSubmission"} {
		for _, tc := range []struct {
			name         string
			filter, auto []Condition
			want         []string
		}{
			{"filter only", []Condition{condition("a")}, nil, []string{"a"}},
			{"auto-event only", nil, []Condition{condition("b")}, []string{"b"}},
			{"both", []Condition{condition("a")}, []Condition{condition("b")}, []string{"a", "b"}},
		} {
			t.Run(triggerType+"/"+tc.name, func(t *testing.T) {
				var sent tagmanager.Trigger
				client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"triggerId":"3"}`)
				})
				input := &TriggerInput{Name: "test", Type: triggerType, Filter: tc.filter, AutoEventFilter: tc.auto}
				if _, err := client.CreateTrigger(context.Background(), "1", "2", "3", input); err != nil {
					t.Fatal(err)
				}
				var got []string
				for _, c := range sent.Filter {
					got = append(got, c.Parameter[0].Value)
				}
				if !slices.Equal(got, tc.want) {
					t.Errorf("filter values = %v, want %v", got, tc.want)
				}
				if len(sent.AutoEventFilter) != 0 {
					t.Errorf("autoEventFilter was sent and would be dropped: %d conditions", len(sent.AutoEventFilter))
				}
			})
		}
	}
}
