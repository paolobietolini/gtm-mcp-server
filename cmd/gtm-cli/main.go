// Command gtm-cli gives people and agents direct access to Google Tag Manager,
// without a server to host.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/paolobietolini/gtm-mcp-server/auth"
	"github.com/paolobietolini/gtm-mcp-server/internal/cli"
)

// version is set by release builds. Builds made with go install report the
// module version instead.
var version string

// timeout bounds a whole command, including retries, so that an unattended
// caller is never left waiting.
const timeout = 2 * time.Minute

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(ctx, timeout)

	app := &cli.App{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
		Credentials: func(ctx context.Context) (*auth.Credentials, error) {
			return auth.ResolveCredentials(ctx, os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY_JSON"))
		},
		TokenInfoURL: auth.TokenInfoURL,
		Version:      buildVersion(),
	}
	code := app.Run(ctx, os.Args[1:])
	cancel()
	stop()
	os.Exit(code)
}

func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
