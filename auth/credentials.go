package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// TokenInfoURL is Google's endpoint describing an access token.
const TokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

// Credentials is a Google identity usable for the GTM API, with enough detail
// to tell a person which one is in use.
type Credentials struct {
	TokenSource oauth2.TokenSource
	// Source names where the credentials were found.
	Source string
	// Type is the credential file type, or "metadata_server".
	Type string
	// Email is empty when only Google knows it, as for user credentials.
	Email string
	// QuotaProject is the project billed for requests, when one is set.
	QuotaProject string
}

// quotaTokenSource carries the quota project along with the tokens, so a client
// built from the token source alone still bills the right project.
type quotaTokenSource struct {
	oauth2.TokenSource
	project string
}

// QuotaProjectID returns the project to bill for requests.
func (q quotaTokenSource) QuotaProjectID() string { return q.project }

// ResolveCredentials finds Google credentials scoped to GTM API access. If
// keyJSON is non-empty it is parsed as a service account key; otherwise
// Application Default Credentials are used.
func ResolveCredentials(ctx context.Context, keyJSON string) (*Credentials, error) {
	if keyJSON != "" {
		config, err := google.JWTConfigFromJSON([]byte(keyJSON), GoogleScopes...)
		if err != nil {
			return nil, fmt.Errorf("invalid GOOGLE_SERVICE_ACCOUNT_KEY_JSON: %w", err)
		}
		return &Credentials{
			TokenSource: config.TokenSource(ctx),
			Source:      "GOOGLE_SERVICE_ACCOUNT_KEY_JSON",
			Type:        "service_account",
			Email:       config.Email,
		}, nil
	}

	found, err := google.FindDefaultCredentials(ctx, GoogleScopes...)
	if err != nil {
		return nil, fmt.Errorf("no service account credentials found (set GOOGLE_SERVICE_ACCOUNT_KEY_JSON or GOOGLE_APPLICATION_CREDENTIALS, or use Workload Identity): %w", err)
	}

	creds := &Credentials{TokenSource: found.TokenSource}
	var file struct {
		Type             string `json:"type"`
		ClientEmail      string `json:"client_email"`
		QuotaProject     string `json:"quota_project_id"`
		ImpersonationURL string `json:"service_account_impersonation_url"`
	}
	switch {
	case len(found.JSON) == 0:
		creds.Source, creds.Type = "metadata server", "metadata_server"
	default:
		if err := json.Unmarshal(found.JSON, &file); err != nil {
			return nil, fmt.Errorf("invalid credentials file: %w", err)
		}
		creds.Source = "gcloud application-default credentials"
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			creds.Source = "GOOGLE_APPLICATION_CREDENTIALS"
		}
		creds.Type = file.Type
		creds.Email = file.ClientEmail
		if creds.Email == "" {
			creds.Email = impersonatedEmail(file.ImpersonationURL)
		}
	}

	// The environment variable wins over the file, as in Google's libraries.
	creds.QuotaProject = os.Getenv("GOOGLE_CLOUD_QUOTA_PROJECT")
	if creds.QuotaProject == "" {
		creds.QuotaProject = file.QuotaProject
	}
	if creds.QuotaProject != "" {
		creds.TokenSource = quotaTokenSource{TokenSource: creds.TokenSource, project: creds.QuotaProject}
	}
	return creds, nil
}

// impersonatedEmail extracts the account from an impersonation URL such as
// ".../serviceAccounts/name@project.iam.gserviceaccount.com:generateAccessToken".
func impersonatedEmail(impersonationURL string) string {
	_, after, found := strings.Cut(impersonationURL, "/serviceAccounts/")
	if !found {
		return ""
	}
	email, _, _ := strings.Cut(after, ":")
	return email
}

// TokenDetails is what Google reports about an access token.
type TokenDetails struct {
	Email     string
	Scopes    []string
	ExpiresIn int
}

// InspectToken asks the tokeninfo endpoint about an access token. The token is
// sent in the request body so that it never appears in a URL.
func InspectToken(ctx context.Context, endpoint, accessToken string) (*TokenDetails, error) {
	form := url.Values{"access_token": {accessToken}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token inspection failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token inspection failed: %s", resp.Status)
	}

	var body struct {
		Email     string      `json:"email"`
		Scope     string      `json:"scope"`
		ExpiresIn json.Number `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return nil, fmt.Errorf("invalid token inspection response: %w", err)
	}
	expiresIn, _ := strconv.Atoi(body.ExpiresIn.String())
	return &TokenDetails{Email: body.Email, Scopes: strings.Fields(body.Scope), ExpiresIn: expiresIn}, nil
}

// MissingScopes returns the GTM scopes that are not in granted.
func MissingScopes(granted []string) []string {
	var missing []string
	for _, scope := range GoogleScopes {
		if !slices.Contains(granted, scope) {
			missing = append(missing, scope)
		}
	}
	return missing
}
