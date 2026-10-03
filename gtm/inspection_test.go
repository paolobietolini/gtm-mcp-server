package gtm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestGetContainerVersion(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/versions/7" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"accountId":"1","containerId":"2","containerVersionId":"7",
			"name":"Release 7","description":"tested","fingerprint":"fp7",
			"path":"accounts/1/containers/2/versions/7",
			"container":{"containerId":"2","name":"Web","publicId":"GTM-ABC","tagIds":["G-123"],"domainName":["example.com"]},
			"tag":[{"tagId":"10","name":"GA4","type":"googtag"}],
			"trigger":[{"triggerId":"20","name":"All pages","type":"pageview"}]
		}`)
	})

	got, err := client.GetContainerVersion(context.Background(), "1", "2", "7")
	if err != nil {
		t.Fatal(err)
	}
	if got.VersionID != "7" || got.Name != "Release 7" || got.Fingerprint != "fp7" {
		t.Fatalf("unexpected version metadata: %+v", got)
	}
	if got.Container == nil || got.Container.PublicID != "GTM-ABC" || len(got.Container.TagIDs) != 1 {
		t.Fatalf("unexpected container: %+v", got.Container)
	}
	if got.Tags == nil || got.Triggers == nil {
		t.Fatalf("version entities were lost: tags=%v triggers=%v", got.Tags, got.Triggers)
	}
	if got.Variables != nil {
		t.Fatalf("empty entity collection should be omitted: %v", got.Variables)
	}
}

func TestGetLiveContainerVersion(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/versions:live" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"containerVersionId":"9","name":"Published","path":"accounts/1/containers/2/versions/9"}`)
	})

	got, err := client.GetLiveContainerVersion(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	if got.VersionID != "9" || got.Name != "Published" {
		t.Fatalf("unexpected live version: %+v", got)
	}
}

func TestLookupContainer(t *testing.T) {
	tests := []struct {
		name          string
		destinationID string
		tagID         string
		wantQuery     url.Values
	}{
		{name: "destination ID", destinationID: "AW-123", wantQuery: url.Values{"destinationId": {"AW-123"}}},
		{name: "tag ID", tagID: "GTM-ABC", wantQuery: url.Values{"tagId": {"GTM-ABC"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/containers:lookup" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				query := r.URL.Query()
				for key, values := range tt.wantQuery {
					if query.Get(key) != values[0] {
						t.Errorf("query %s=%q want %q", key, query.Get(key), values[0])
					}
				}
				if query.Get("destinationId") != "" && query.Get("tagId") != "" {
					t.Errorf("lookup sent both identifiers: %s", query.Encode())
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"containerId":"2","name":"Found","publicId":"GTM-ABC","tagIds":["AW-123"],"taggingServerUrls":["https://sgtm.example"]}`)
			})

			got, err := client.LookupContainer(context.Background(), tt.destinationID, tt.tagID)
			if err != nil {
				t.Fatal(err)
			}
			if got.ContainerID != "2" || len(got.TagIDs) != 1 || len(got.TaggingServerURLs) != 1 {
				t.Fatalf("unexpected container: %+v", got)
			}
		})
	}
}

func TestLookupContainerRequiresExactlyOneIdentifier(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("validation should prevent request: %s", r.URL)
	})
	for _, args := range [][2]string{{"", ""}, {"AW-123", "GTM-ABC"}, {" ", " "}} {
		if _, err := client.LookupContainer(context.Background(), args[0], args[1]); err == nil {
			t.Fatalf("LookupContainer(%q, %q) succeeded", args[0], args[1])
		}
	}
}

func TestGetContainerSnippet(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2:snippet" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"snippet":"<script>install</script>","containerConfig":"config-data"}`)
	})

	got, err := client.GetContainerSnippet(context.Background(), "1", "2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Snippet != "<script>install</script>" || got.ContainerConfig != "config-data" {
		t.Fatalf("unexpected snippet: %+v", got)
	}
}

func TestInspectionErrors(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":404,"message":"missing"}}`)
	})

	checks := []struct {
		name string
		call func() error
	}{
		{name: "get version", call: func() error { _, err := client.GetContainerVersion(context.Background(), "1", "2", "7"); return err }},
		{name: "live version", call: func() error { _, err := client.GetLiveContainerVersion(context.Background(), "1", "2"); return err }},
		{name: "lookup", call: func() error { _, err := client.LookupContainer(context.Background(), "", "GTM-ABC"); return err }},
		{name: "snippet", call: func() error { _, err := client.GetContainerSnippet(context.Background(), "1", "2"); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, ErrNotFound) {
				t.Fatalf("error=%v, want ErrNotFound", err)
			}
		})
	}
}
