package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

const userADC = `{"type":"authorized_user","client_id":"id","client_secret":"secret","refresh_token":"refresh","quota_project_id":"my-project"}`

func isolateCredentialEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("GOOGLE_CLOUD_QUOTA_PROJECT", "")
	return home
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func quotaProjectOf(c *Credentials) string {
	if q, ok := c.TokenSource.(interface{ QuotaProjectID() string }); ok {
		return q.QuotaProjectID()
	}
	return ""
}

func TestResolveCredentialsFromKeyJSON(t *testing.T) {
	isolateCredentialEnv(t)
	c, err := ResolveCredentials(context.Background(), string(testServiceAccountJSON(t)))
	if err != nil {
		t.Fatal(err)
	}
	if c.Source != "GOOGLE_SERVICE_ACCOUNT_KEY_JSON" || c.Type != "service_account" ||
		c.Email != "gtm-bot@test-project.iam.gserviceaccount.com" || c.QuotaProject != "" {
		t.Fatalf("unexpected credentials: %+v", c)
	}
}

func TestResolveCredentialsFromCredentialsFile(t *testing.T) {
	home := isolateCredentialEnv(t)
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", writeFile(t, filepath.Join(home, "adc.json"), userADC))
	c, err := ResolveCredentials(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Source != "GOOGLE_APPLICATION_CREDENTIALS" || c.Type != "authorized_user" || c.Email != "" {
		t.Fatalf("unexpected credentials: %+v", c)
	}
	if c.QuotaProject != "my-project" || quotaProjectOf(c) != "my-project" {
		t.Fatalf("quota project lost: %q / %q", c.QuotaProject, quotaProjectOf(c))
	}
}

func TestResolveCredentialsFromGcloudImpersonation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("gcloud keeps its file under APPDATA on Windows")
	}
	home := isolateCredentialEnv(t)
	writeFile(t, filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"),
		`{"type":"impersonated_service_account","service_account_impersonation_url":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/gtm@p.iam.gserviceaccount.com:generateAccessToken","source_credentials":`+userADC+`}`)
	c, err := ResolveCredentials(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Source != "gcloud application-default credentials" || c.Type != "impersonated_service_account" ||
		c.Email != "gtm@p.iam.gserviceaccount.com" {
		t.Fatalf("unexpected credentials: %+v", c)
	}
}

func TestInspectToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.PostFormValue("access_token") != "tok" {
			t.Errorf("token must be sent in the POST body: %s %s", r.Method, r.URL)
		}
		w.Write([]byte(`{"email":"me@example.com","scope":"a b","expires_in":"3599"}`))
	}))
	defer server.Close()
	got, err := InspectToken(context.Background(), server.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if got.Email != "me@example.com" || !slices.Equal(got.Scopes, []string{"a", "b"}) || got.ExpiresIn != 3599 {
		t.Fatalf("unexpected details: %+v", got)
	}
}

func TestInspectTokenRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"invalid_token"}`))
	}))
	defer server.Close()
	if _, err := InspectToken(context.Background(), server.URL, "tok"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestMissingScopes(t *testing.T) {
	if got := MissingScopes(GoogleScopes); len(got) != 0 {
		t.Fatalf("expected none, got %v", got)
	}
	if got := MissingScopes(GoogleScopes[1:]); !slices.Equal(got, GoogleScopes[:1]) {
		t.Fatalf("unexpected: %v", got)
	}
}
