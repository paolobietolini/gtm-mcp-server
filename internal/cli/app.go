// Package cli implements the gtm command-line interface. It parses arguments,
// calls gtm.Client and prints the result. Safety rules and API logic live in
// package gtm, shared with the MCP server.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/paolobietolini/gtm-mcp-server/auth"
	"github.com/paolobietolini/gtm-mcp-server/gtm"
)

// App holds everything the CLI takes from its environment.
type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Getenv func(string) string
	// Credentials finds the Google identity to act as.
	Credentials func(context.Context) (*auth.Credentials, error)
	// Endpoint overrides the Tag Manager API base URL. Empty means Google's.
	Endpoint     string
	TokenInfoURL string
	Version      string
}

// options are the flag values of one invocation.
type options struct {
	account, container, workspace string
	id                            string
	file                          string
	name, notes                   string
	output                        string
	confirm, dryRun               bool
}

// Run executes one command and returns the process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	if err := a.run(ctx, args); err != nil {
		return a.report(err)
	}
	return 0
}

func (a *App) run(ctx context.Context, args []string) error {
	if len(args) == 0 || isHelp(args[0]) || args[0] == "help" {
		a.usage("")
		return nil
	}
	if args[0] == "version" || args[0] == "--version" {
		if len(args) > 1 {
			return usageError("version takes no arguments")
		}
		return a.printJSON(map[string]string{"version": a.Version})
	}

	noun := args[0]
	if !knownNoun(noun) {
		return usageError("unknown command %q; flags go after the command, see `gtm help`", noun)
	}
	if len(args) == 1 || isHelp(args[1]) {
		a.usage(noun)
		return nil
	}
	cmd := findCommand(noun, args[1])
	if cmd == nil {
		return usageError("unknown command %q, see `gtm %s --help`", noun+" "+args[1], noun)
	}
	return a.execute(ctx, cmd, args[2:])
}

func (a *App) execute(ctx context.Context, cmd *command, args []string) error {
	var opts options
	fs := cmd.flagSet(&opts)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(a.Stdout, "gtm %s %s: %s\n\nFlags:\n", cmd.noun, cmd.verb, cmd.summary)
			fs.SetOutput(a.Stdout)
			fs.PrintDefaults()
			return nil
		}
		return usageError("%v", err)
	}
	if fs.NArg() > 0 {
		return usageError("unexpected argument %q; pass values with flags", fs.Arg(0))
	}
	if opts.output != "json" && opts.output != "table" {
		return usageError("--output must be json or table")
	}

	if err := a.resolveIDs(cmd, &opts); err != nil {
		return err
	}
	// A dry run sends nothing, so it needs no confirmation.
	if cmd.confirm != "" && !opts.dryRun {
		if err := gtm.CheckConfirm(cmd.confirm, opts.confirm); err != nil {
			return err
		}
	}

	inv := &invocation{options: opts}
	if cmd.takesFile {
		data, err := a.readInput(opts.file)
		if err != nil {
			return err
		}
		inv.data = data
	}
	// Reject bad input before looking for credentials or contacting Google.
	if cmd.prepare != nil {
		if err := cmd.prepare(inv); err != nil {
			return inputError(err)
		}
	}

	if cmd.local != nil {
		return cmd.local(ctx, a, inv)
	}

	client, err := a.client(ctx, opts.dryRun)
	if err != nil {
		return err
	}
	out, err := cmd.run(ctx, client, inv)
	var intercepted *dryRunError
	if errors.As(err, &intercepted) {
		return a.printJSON(dryRunOutput{DryRun: true, Request: intercepted.request})
	}
	if err != nil {
		return err
	}
	if opts.output == "table" {
		return a.printTable(out, cmd.columns)
	}
	return a.printJSON(out)
}

func (a *App) readInput(path string) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(a.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, inputError(fmt.Errorf("cannot read input: %w", err))
	}
	return data, nil
}

func isHelp(arg string) bool { return arg == "--help" || arg == "-h" || arg == "-help" }

func (a *App) usage(noun string) {
	var b strings.Builder
	b.WriteString("gtm - Google Tag Manager from the command line\n\nUsage: gtm <noun> <verb> [flags]\n\nCommands:\n")
	for _, cmd := range commands {
		if noun == "" || cmd.noun == noun {
			fmt.Fprintf(&b, "  %-22s %s\n", cmd.noun+" "+cmd.verb, cmd.summary)
		}
	}
	if noun == "" {
		fmt.Fprintf(&b, "  %-22s %s\n", "version", "Print the CLI version")
	}
	b.WriteString("\nRun `gtm <noun> <verb> --help` for flags.\n")
	if noun == "" {
		b.WriteString(`
ID defaults: --account, --container and --workspace fall back to
GTM_ACCOUNT_ID, GTM_CONTAINER_ID and GTM_WORKSPACE_ID, then to the config file
(GTM_CONFIG, or ~/.config/gtm/config.json).

Output is JSON; errors are JSON on stderr.
Exit codes: 0 ok, 1 API or network error, 2 usage or invalid input,
3 authentication, 4 permission denied, 5 not found, 6 conflict or failed
precondition, 7 rate limited, 8 confirmation required.
`)
	}
	fmt.Fprint(a.Stdout, b.String())
}
