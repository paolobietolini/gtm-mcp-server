package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/paolobietolini/gtm-mcp-server/gtm"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// client builds a GTM client. In a dry run, requests that would change anything
// are intercepted, and credentials are only needed if something must be read.
func (a *App) client(ctx context.Context, dryRun bool) (*gtm.Client, error) {
	source := &tokenSource{}
	creds, err := a.Credentials(ctx)
	switch {
	case err == nil:
		source.source, source.quotaProject = creds.TokenSource, creds.QuotaProject
	case dryRun:
		source.err = err
	default:
		return nil, authError(err)
	}

	var opts []option.ClientOption
	if a.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(a.Endpoint))
	}
	wrap := func(next http.RoundTripper) http.RoundTripper {
		if dryRun {
			return dryRunTransport{next}
		}
		return next
	}
	return gtm.NewClientWithTransport(ctx, source, wrap, opts...)
}

// tokenSource marks every failure to obtain a token as an authentication error.
type tokenSource struct {
	source       oauth2.TokenSource
	quotaProject string
	err          error
}

func (s *tokenSource) Token() (*oauth2.Token, error) {
	if s.err != nil {
		return nil, authError(s.err)
	}
	token, err := s.source.Token()
	if err != nil {
		return nil, authError(err)
	}
	return token, nil
}

func (s *tokenSource) QuotaProjectID() string { return s.quotaProject }

// dryRunRequest is the request a write command would have sent.
type dryRunRequest struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Query  map[string]string `json:"query,omitempty"`
	Body   json.RawMessage   `json:"body,omitempty"`
}

type dryRunOutput struct {
	DryRun  bool          `json:"dryRun"`
	Request dryRunRequest `json:"request"`
}

type dryRunError struct{ request dryRunRequest }

func (e *dryRunError) Error() string { return "dry run: request not sent" }

// dryRunTransport lets reads through and stops everything else, so a dry run
// shows exactly what the API client built.
type dryRunTransport struct{ next http.RoundTripper }

func (t dryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet {
		return t.next.RoundTrip(req)
	}
	request := dryRunRequest{Method: req.Method}
	if req.Body != nil {
		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if json.Valid(body) {
			request.Body = body
		}
	}
	url := *req.URL
	for key, values := range url.Query() {
		// Added by the API library to every request.
		if key == "alt" || key == "prettyPrint" || len(values) == 0 {
			continue
		}
		if request.Query == nil {
			request.Query = map[string]string{}
		}
		request.Query[key] = values[0]
	}
	url.RawQuery = ""
	request.URL = url.String()
	return nil, &dryRunError{request}
}
