package gtm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

func versionTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := tagmanager.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return &Client{Service: service}
}

func TestLatestVersionHeader(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/version_headers:latest" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"containerVersionId":"8","name":"Draft snapshot","numTags":"3","path":"accounts/1/containers/2/versions/8"}`)
	})
	got, err := client.LatestVersionHeader(context.Background(), "accounts/1/containers/2")
	if err != nil {
		t.Fatal(err)
	}
	if got.VersionID != "8" || got.NumTags != "3" || got.Name != "Draft snapshot" {
		t.Fatalf("unexpected header: %+v", got)
	}
}

func TestListVersionHeadersPagination(t *testing.T) {
	calls := 0
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/tagmanager/v2/accounts/1/containers/2/version_headers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"containerVersionHeader":[{"containerVersionId":"1"}],"nextPageToken":"page2"}`)
		case "page2":
			fmt.Fprint(w, `{"containerVersionHeader":[{"containerVersionId":"2"}]}`)
		default:
			t.Error("unexpected page token")
			w.WriteHeader(400)
		}
	})
	got, err := client.ListVersionHeaders(context.Background(), "accounts/1/containers/2")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got) != 2 || got[0].VersionID != "1" || got[1].VersionID != "2" {
		t.Fatalf("calls=%d versions=%+v", calls, got)
	}
}

func TestVersionHeaderErrors(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":404,"message":"missing container"}}`)
	})
	if _, err := client.LatestVersionHeader(context.Background(), "accounts/1/containers/2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("latest: %v", err)
	}
	if _, err := client.ListVersionHeaders(context.Background(), "accounts/1/containers/2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("list: %v", err)
	}
}

func TestListVersionHeadersEmpty(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	})
	got, err := client.ListVersionHeaders(context.Background(), "accounts/1/containers/2")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("versions=%v err=%v", got, err)
	}
}

func TestListVersionHeadersLaterPageError(t *testing.T) {
	client := versionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("pageToken") == "" {
			fmt.Fprint(w, `{"containerVersionHeader":[{"containerVersionId":"1"}],"nextPageToken":"next"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error":{"code":404,"message":"container removed"}}`)
		}
	})
	got, err := client.ListVersionHeaders(context.Background(), "accounts/1/containers/2")
	if !errors.Is(err, ErrNotFound) || got != nil {
		t.Fatalf("partial results reported as complete: %v, %v", got, err)
	}
}
