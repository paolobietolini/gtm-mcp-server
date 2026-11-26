package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/paolobietolini/gtm-mcp-server/gtm"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

// Exit codes are part of the CLI's interface: scripts and agents branch on them.
const (
	exitError        = 1
	exitUsage        = 2
	exitAuth         = 3
	exitPermission   = 4
	exitNotFound     = 5
	exitConflict     = 6
	exitRateLimit    = 7
	exitConfirmation = 8
)

const authHint = "`gtm-cli auth status` shows the identity in use. For Tag Manager scopes, log in with " +
	"your own OAuth client (gcloud auth application-default login --client-id-file=...) or use a " +
	"service account; see https://github.com/paolobietolini/gtm-mcp-server#command-line-interface"

// cliError is a failure the CLI itself classified.
type cliError struct {
	code string
	exit int
	err  error
}

func (e *cliError) Error() string { return e.err.Error() }
func (e *cliError) Unwrap() error { return e.err }

func usageError(format string, args ...any) error {
	return &cliError{"usage", exitUsage, fmt.Errorf(format, args...)}
}

func inputError(err error) error {
	var classified *cliError
	if errors.As(err, &classified) {
		return err
	}
	return &cliError{"invalid_input", exitUsage, err}
}

func authError(err error) error { return &cliError{"auth", exitAuth, err} }

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// report writes err to stderr as JSON and returns the exit code.
func (a *App) report(err error) int {
	body, exit := classify(err)
	encoder := json.NewEncoder(a.Stderr)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	// Nothing useful can be done if stderr itself fails.
	_ = encoder.Encode(map[string]errorBody{"error": body})
	return exit
}

func classify(err error) (errorBody, int) {
	body := errorBody{Message: err.Error()}
	result := func(code string, exit int) (errorBody, int) {
		body.Code = code
		if code == "auth" {
			body.Hint = authHint
		}
		return body, exit
	}

	var own *cliError
	var blocked *gtm.VersionBlockedError
	var refresh *oauth2.RetrieveError
	var apiErr *googleapi.Error
	switch {
	case errors.As(err, &own):
		// Drop the HTTP client's wrapping around a token failure.
		body.Message = own.Error()
		return result(own.code, own.exit)
	case errors.Is(err, gtm.ErrConfirmationRequired):
		body.Message = strings.Replace(body.Message, "confirm: true", "--confirm", 1)
		body.Hint = "Preview with --dry-run, get approval, then repeat with --confirm."
		return result("confirmation_required", exitConfirmation)
	case errors.As(err, &blocked):
		return result("precondition_failed", exitConflict)
	case errors.As(err, &refresh):
		return result("auth", exitAuth)
	case errors.Is(err, gtm.ErrPermission):
		// These 403s are fixed by changing credentials, not GTM permissions.
		if strings.Contains(body.Message, "insufficient authentication scopes") || strings.Contains(body.Message, "quota project") {
			return result("auth", exitAuth)
		}
		return result("permission_denied", exitPermission)
	case errors.Is(err, gtm.ErrNotFound):
		return result("not_found", exitNotFound)
	case errors.Is(err, gtm.ErrConflict):
		return result("conflict", exitConflict)
	case errors.Is(err, gtm.ErrRateLimit):
		return result("rate_limited", exitRateLimit)
	case errors.Is(err, gtm.ErrInvalidRequest):
		return result("invalid_request", exitUsage)
	case errors.As(err, &apiErr):
		if apiErr.Code == http.StatusUnauthorized {
			return result("auth", exitAuth)
		}
		return result("api_error", exitError)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return result("cancelled", exitError)
	default:
		return result("error", exitError)
	}
}
