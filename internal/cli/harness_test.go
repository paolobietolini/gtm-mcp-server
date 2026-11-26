package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/paolobietolini/gtm-mcp-server/auth"

	"golang.org/x/oauth2"
)

type recorded struct {
	method, path, query, body string
}

// fakeAPI records every request and answers from replies, keyed by
// "METHOD /path". Paths omit the /tagmanager/v2/ prefix.
type fakeAPI struct {
	mu       sync.Mutex
	requests []recorded
	replies  map[string]reply
}

type reply struct {
	status int
	body   string
}

type result struct {
	exit           int
	stdout, stderr string
}

type harness struct {
	t     *testing.T
	api   *fakeAPI
	app   *App
	env   map[string]string
	stdin string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	api := &fakeAPI{replies: map[string]reply{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		path := strings.TrimPrefix(r.URL.Path, "/tagmanager/v2/")
		api.mu.Lock()
		api.requests = append(api.requests, recorded{r.Method, path, r.URL.RawQuery, string(body)})
		answer, ok := api.replies[r.Method+" "+path]
		api.mu.Unlock()
		if !ok {
			t.Errorf("unexpected request: %s %s", r.Method, path)
			answer = reply{404, `{"error":{"code":404,"message":"no reply configured"}}`}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(answer.status)
		fmt.Fprint(w, answer.body)
	}))
	t.Cleanup(server.Close)

	h := &harness{t: t, api: api, env: map[string]string{"HOME": t.TempDir()}}
	h.app = &App{
		Getenv: func(key string) string { return h.env[key] },
		Credentials: func(context.Context) (*auth.Credentials, error) {
			return &auth.Credentials{
				TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "tok"}),
				Source:      "test",
				Type:        "service_account",
				Email:       "bot@example.com",
			}, nil
		},
		Endpoint:     server.URL + "/",
		TokenInfoURL: server.URL + "/tokeninfo",
		Version:      "1.2.3",
	}
	return h
}

// withWorkspace sets the ID defaults to account 1, container 2, workspace 3.
func (h *harness) withWorkspace() *harness {
	h.env["GTM_ACCOUNT_ID"], h.env["GTM_CONTAINER_ID"], h.env["GTM_WORKSPACE_ID"] = "1", "2", "3"
	return h
}

func (h *harness) on(request, body string) *harness { return h.onStatus(request, 200, body) }

func (h *harness) onStatus(request string, status int, body string) *harness {
	h.api.replies[request] = reply{status, body}
	return h
}

func (h *harness) noCredentials() *harness {
	h.app.Credentials = func(context.Context) (*auth.Credentials, error) {
		return nil, errors.New("could not find default credentials")
	}
	return h
}

func (h *harness) run(args ...string) result {
	h.t.Helper()
	var stdout, stderr bytes.Buffer
	h.app.Stdin, h.app.Stdout, h.app.Stderr = strings.NewReader(h.stdin), &stdout, &stderr
	exit := h.app.Run(context.Background(), args)
	return result{exit, stdout.String(), stderr.String()}
}

// ok runs a command that must succeed and returns its decoded JSON output.
func (h *harness) ok(args ...string) map[string]any {
	h.t.Helper()
	got := h.run(args...)
	if got.exit != 0 || got.stderr != "" {
		h.t.Fatalf("gtm %s: exit=%d stderr=%s", strings.Join(args, " "), got.exit, got.stderr)
	}
	return decodeObject(h.t, got.stdout)
}

// fails runs a command that must fail with the given exit code and error code,
// reporting the error as JSON on stderr.
func (h *harness) fails(exit int, code string, args ...string) string {
	h.t.Helper()
	got := h.run(args...)
	if got.exit != exit {
		h.t.Fatalf("gtm %s: exit=%d want %d; stdout=%s stderr=%s", strings.Join(args, " "), got.exit, exit, got.stdout, got.stderr)
	}
	failure, _ := decodeObject(h.t, got.stderr)["error"].(map[string]any)
	if failure["code"] != code {
		h.t.Fatalf("gtm %s: code=%v want %s; stderr=%s", strings.Join(args, " "), failure["code"], code, got.stderr)
	}
	message, _ := failure["message"].(string)
	return message
}

func (h *harness) sent() []recorded {
	h.api.mu.Lock()
	defer h.api.mu.Unlock()
	return append([]recorded(nil), h.api.requests...)
}

func (h *harness) noRequests() {
	h.t.Helper()
	if got := h.sent(); len(got) != 0 {
		h.t.Fatalf("expected no requests, got %+v", got)
	}
}

func decodeObject(t *testing.T, text string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatalf("not a JSON object: %v\n%s", err, text)
	}
	return value
}

// at walks nested objects, for assertions such as at(out, "tag", "name").
func at(value any, path ...string) any {
	for _, key := range path {
		object, _ := value.(map[string]any)
		value = object[key]
	}
	return value
}
