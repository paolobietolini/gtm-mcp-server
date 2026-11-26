package cli

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/paolobietolini/gtm-mcp-server/auth"
)

const tagManagerScopePrefix = "https://www.googleapis.com/auth/tagmanager."

type authStatusOutput struct {
	Authenticated bool     `json:"authenticated"`
	Source        string   `json:"source"`
	Type          string   `json:"type"`
	Email         string   `json:"email,omitempty"`
	QuotaProject  string   `json:"quotaProject,omitempty"`
	Scopes        []string `json:"scopes"`
	MissingScopes []string `json:"missingScopes"`
	ExpiresIn     int      `json:"expiresIn,omitempty"`
}

// authStatus reports the identity in use, after proving that it can obtain a
// token.
func authStatus(ctx context.Context, a *App, in *invocation) error {
	creds, err := a.Credentials(ctx)
	if err != nil {
		return authError(err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return authError(err)
	}
	details, err := auth.InspectToken(ctx, a.TokenInfoURL, token.AccessToken)
	if err != nil {
		return authError(err)
	}

	out := authStatusOutput{
		Authenticated: true,
		Source:        creds.Source,
		Type:          creds.Type,
		Email:         details.Email,
		QuotaProject:  creds.QuotaProject,
		Scopes:        append([]string{}, details.Scopes...),
		MissingScopes: append([]string{}, auth.MissingScopes(details.Scopes)...),
		ExpiresIn:     details.ExpiresIn,
	}
	if out.Email == "" {
		out.Email = creds.Email
	}

	if in.output == "table" {
		err = a.printTable(out, nil)
	} else {
		err = a.printJSON(out)
	}
	if err != nil {
		return err
	}
	usable := slices.ContainsFunc(details.Scopes, func(scope string) bool {
		return strings.HasPrefix(scope, tagManagerScopePrefix)
	})
	if !usable {
		return authError(errors.New("these credentials have no Tag Manager scope"))
	}
	return nil
}
