package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/paolobietolini/gtm-mcp-server/gtm"
)

// level is how far down the GTM hierarchy a command operates, and so which IDs
// it needs.
type level int

const (
	levelNone level = iota
	levelAccount
	levelContainer
	levelWorkspace
)

// invocation is a parsed command ready to run.
type invocation struct {
	options
	data []byte
	// input is the decoded --file content, set by prepare.
	input any
}

type command struct {
	noun, verb string
	summary    string
	level      level
	// idFlag names the flag selecting one entity, such as "tag-id".
	idFlag    string
	takesFile bool
	// writes marks commands that change something and so support --dry-run.
	writes bool
	// confirm is the guarded operation, empty if no confirmation is needed.
	confirm gtm.Operation
	flags   func(*flag.FlagSet, *options)
	// columns are the JSON keys shown by table output for lists.
	columns []string
	prepare func(*invocation) error
	run     func(context.Context, *gtm.Client, *invocation) (any, error)
	// local replaces run for commands that do not use the GTM API.
	local func(context.Context, *App, *invocation) error
}

func (cmd *command) flagSet(opts *options) *flag.FlagSet {
	fs := flag.NewFlagSet("gtm-cli "+cmd.noun+" "+cmd.verb, flag.ContinueOnError)
	// Errors are reported as JSON, so keep the flag package quiet.
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.output, "output", "json", "output format: json or table")
	if cmd.level >= levelAccount {
		fs.StringVar(&opts.account, "account", "", "GTM account ID (default $GTM_ACCOUNT_ID)")
	}
	if cmd.level >= levelContainer {
		fs.StringVar(&opts.container, "container", "", "GTM container ID (default $GTM_CONTAINER_ID)")
	}
	if cmd.level >= levelWorkspace {
		fs.StringVar(&opts.workspace, "workspace", "", "GTM workspace ID (default $GTM_WORKSPACE_ID)")
	}
	if cmd.idFlag != "" {
		fs.StringVar(&opts.id, cmd.idFlag, "", "ID of the entity (required)")
	}
	if cmd.takesFile {
		fs.StringVar(&opts.file, "file", "", "JSON input file, or - for stdin (required)")
	}
	if cmd.writes {
		fs.BoolVar(&opts.dryRun, "dry-run", false, "print the request that would be sent, without sending it")
	}
	if cmd.confirm != "" {
		fs.BoolVar(&opts.confirm, "confirm", false, "confirm this destructive operation")
	}
	if cmd.flags != nil {
		cmd.flags(fs, opts)
	}
	return fs
}

func knownNoun(noun string) bool {
	for _, cmd := range commands {
		if cmd.noun == noun {
			return true
		}
	}
	return false
}

func findCommand(noun, verb string) *command {
	for _, cmd := range commands {
		if cmd.noun == noun && cmd.verb == verb {
			return cmd
		}
	}
	return nil
}

// commands mirror the MCP tools: the tool verb_noun is `gtm-cli <nouns> <verb>`,
// and the output has the same shape as the tool's.
var commands = buildCommands()

func buildCommands() []*command {
	list := []*command{
		{noun: "auth", verb: "status", summary: "Show the Google identity in use", local: authStatus},
		{
			noun: "accounts", verb: "list", summary: "List GTM accounts",
			columns: []string{"accountId", "name"},
			run: func(ctx context.Context, c *gtm.Client, _ *invocation) (any, error) {
				accounts, err := c.ListAccounts(ctx)
				return gtm.ListAccountsOutput{Accounts: accounts}, err
			},
		},
		{
			noun: "containers", verb: "list", summary: "List the containers of an account", level: levelAccount,
			columns: []string{"containerId", "name", "publicId", "usageContext"},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				containers, err := c.ListContainers(ctx, in.account)
				return gtm.ListContainersOutput{Containers: containers}, err
			},
		},
		{
			noun: "workspaces", verb: "list", summary: "List the workspaces of a container", level: levelContainer,
			columns: []string{"workspaceId", "name", "description"},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				workspaces, err := c.ListWorkspaces(ctx, in.account, in.container)
				return gtm.ListWorkspacesOutput{Workspaces: workspaces}, err
			},
		},
		{
			noun: "workspaces", verb: "status", summary: "Show pending changes and conflicts", level: levelWorkspace,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				status, err := c.GetWorkspaceStatus(ctx, in.account, in.container, in.workspace)
				if err != nil {
					return nil, err
				}
				return gtm.GetWorkspaceStatusOutput{Status: *status}, nil
			},
		},
	}
	list = append(list, tagCommands()...)
	list = append(list, triggerCommands()...)
	list = append(list, variableCommands()...)
	return append(list, versionCommands()...)
}

func tagCommands() []*command {
	return []*command{
		{
			noun: "tags", verb: "list", summary: "List tags", level: levelWorkspace,
			columns: []string{"tagId", "name", "type", "paused"},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				tags, err := c.ListTags(ctx, in.account, in.container, in.workspace)
				return gtm.ListTagsOutput{Tags: tags}, err
			},
		},
		{
			noun: "tags", verb: "get", summary: "Show one tag", level: levelWorkspace, idFlag: "tag-id",
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				tag, err := c.GetTag(ctx, in.account, in.container, in.workspace, in.id)
				if err != nil {
					return nil, err
				}
				return gtm.GetTagOutput{Tag: *tag}, nil
			},
		},
		{
			noun: "tags", verb: "create", summary: "Create a tag", level: levelWorkspace, takesFile: true, writes: true,
			prepare: func(in *invocation) error {
				input, err := decodeTagInput(in.data)
				if err != nil {
					return err
				}
				in.input = input
				return gtm.ValidateTagInput(input.Name, input.Type, input.FiringTriggerId)
			},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				tag, err := c.CreateTag(ctx, in.account, in.container, in.workspace, in.input.(*gtm.TagInput))
				if err != nil {
					return nil, err
				}
				return gtm.CreateTagOutput{Success: true, Tag: *tag, Message: "Tag created successfully"}, nil
			},
		},
		{
			noun: "tags", verb: "update", summary: "Update a tag; omitted fields are kept", level: levelWorkspace, idFlag: "tag-id", takesFile: true, writes: true,
			prepare: func(in *invocation) error {
				input, err := decodeTagInput(in.data)
				in.input = input
				return err
			},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				path := gtm.BuildTagPath(in.account, in.container, in.workspace, in.id)
				tag, err := c.UpdateTag(ctx, path, in.input.(*gtm.TagInput))
				if err != nil {
					return nil, err
				}
				return gtm.UpdateTagOutput{Success: true, Tag: *tag, Message: "Tag updated successfully"}, nil
			},
		},
		{
			noun: "tags", verb: "delete", summary: "Delete a tag", level: levelWorkspace, idFlag: "tag-id", writes: true, confirm: gtm.OpDeleteTag,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				if err := c.DeleteTag(ctx, gtm.BuildTagPath(in.account, in.container, in.workspace, in.id)); err != nil {
					return nil, err
				}
				return gtm.DeleteTagOutput{Success: true, Message: fmt.Sprintf("Tag %s deleted successfully", in.id)}, nil
			},
		},
	}
}

func triggerCommands() []*command {
	prepare := func(in *invocation) error {
		input, err := decodeTriggerInput(in.data)
		if err != nil {
			return err
		}
		in.input = input
		return gtm.ValidateTriggerInput(input.Name, input.Type)
	}
	return []*command{
		{
			noun: "triggers", verb: "list", summary: "List triggers", level: levelWorkspace,
			columns: []string{"triggerId", "name", "type"},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				triggers, err := c.ListTriggers(ctx, in.account, in.container, in.workspace)
				return gtm.ListTriggersOutput{Triggers: triggers}, err
			},
		},
		{
			noun: "triggers", verb: "get", summary: "Show one trigger", level: levelWorkspace, idFlag: "trigger-id",
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				trigger, err := c.GetTrigger(ctx, in.account, in.container, in.workspace, in.id)
				if err != nil {
					return nil, err
				}
				return gtm.GetTriggerOutput{Trigger: *trigger}, nil
			},
		},
		{
			noun: "triggers", verb: "create", summary: "Create a trigger", level: levelWorkspace, takesFile: true, writes: true,
			prepare: prepare,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				input := in.input.(*gtm.TriggerInput)
				trigger, err := c.CreateTrigger(ctx, in.account, in.container, in.workspace, input)
				if err != nil {
					return nil, err
				}
				return gtm.CreateTriggerOutput{Success: true, Trigger: *trigger, Message: gtm.TriggerCreateMessage(input)}, nil
			},
		},
		{
			noun: "triggers", verb: "update", summary: "Update a trigger; name and type are required", level: levelWorkspace, idFlag: "trigger-id", takesFile: true, writes: true,
			prepare: prepare,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				input := in.input.(*gtm.TriggerInput)
				warning := gtm.NormalizeTriggerUpdate(input)
				path := gtm.BuildTriggerPath(in.account, in.container, in.workspace, in.id)
				trigger, err := c.UpdateTrigger(ctx, path, input)
				if err != nil {
					return nil, err
				}
				message := "Trigger updated successfully"
				if warning != "" {
					message += ". " + warning
				}
				return gtm.UpdateTriggerOutput{Success: true, Trigger: *trigger, Message: message}, nil
			},
		},
		{
			noun: "triggers", verb: "delete", summary: "Delete a trigger", level: levelWorkspace, idFlag: "trigger-id", writes: true, confirm: gtm.OpDeleteTrigger,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				if err := c.DeleteTrigger(ctx, gtm.BuildTriggerPath(in.account, in.container, in.workspace, in.id)); err != nil {
					return nil, err
				}
				return gtm.DeleteTriggerOutput{Success: true, Message: fmt.Sprintf("Trigger %s deleted successfully", in.id)}, nil
			},
		},
	}
}

func variableCommands() []*command {
	prepare := func(in *invocation) error {
		input, err := decodeVariableInput(in.data)
		if err != nil {
			return err
		}
		in.input = input
		return gtm.ValidateVariableInput(input.Name, input.Type)
	}
	return []*command{
		{
			noun: "variables", verb: "list", summary: "List variables", level: levelWorkspace,
			columns: []string{"variableId", "name", "type"},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				variables, err := c.ListVariables(ctx, in.account, in.container, in.workspace)
				return gtm.ListVariablesOutput{Variables: variables}, err
			},
		},
		{
			noun: "variables", verb: "get", summary: "Show one variable", level: levelWorkspace, idFlag: "variable-id",
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				variable, err := c.GetVariable(ctx, in.account, in.container, in.workspace, in.id)
				if err != nil {
					return nil, err
				}
				return gtm.GetVariableOutput{Variable: *variable}, nil
			},
		},
		{
			noun: "variables", verb: "create", summary: "Create a variable", level: levelWorkspace, takesFile: true, writes: true,
			prepare: prepare,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				variable, err := c.CreateVariable(ctx, in.account, in.container, in.workspace, in.input.(*gtm.VariableInput))
				if err != nil {
					return nil, err
				}
				return gtm.CreateVariableOutput{Success: true, Variable: *variable, Message: "Variable created successfully"}, nil
			},
		},
		{
			noun: "variables", verb: "update", summary: "Update a variable; name and type are required", level: levelWorkspace, idFlag: "variable-id", takesFile: true, writes: true,
			prepare: prepare,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				path := gtm.BuildVariablePath(in.account, in.container, in.workspace, in.id)
				variable, err := c.UpdateVariable(ctx, path, in.input.(*gtm.VariableInput))
				if err != nil {
					return nil, err
				}
				return gtm.UpdateVariableOutput{Success: true, Variable: *variable, Message: "Variable updated successfully"}, nil
			},
		},
		{
			noun: "variables", verb: "delete", summary: "Delete a variable", level: levelWorkspace, idFlag: "variable-id", writes: true, confirm: gtm.OpDeleteVariable,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				if err := c.DeleteVariable(ctx, gtm.BuildVariablePath(in.account, in.container, in.workspace, in.id)); err != nil {
					return nil, err
				}
				return gtm.DeleteVariableOutput{Success: true, Message: fmt.Sprintf("Variable %s deleted successfully", in.id)}, nil
			},
		},
	}
}

func versionCommands() []*command {
	return []*command{
		{
			noun: "versions", verb: "create", summary: "Create a version from workspace changes", level: levelWorkspace, writes: true,
			flags: func(fs *flag.FlagSet, opts *options) {
				fs.StringVar(&opts.name, "name", "", "version name")
				fs.StringVar(&opts.notes, "notes", "", "version notes")
			},
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				version, err := c.CreateVersionFromWorkspace(ctx, in.account, in.container, in.workspace,
					&gtm.VersionInput{Name: in.name, Notes: in.notes})
				if err != nil {
					return nil, err
				}
				return gtm.CreateVersionOutput{Success: true, Version: *version, Message: "Version created successfully. Use `gtm-cli versions publish` to make it live."}, nil
			},
		},
		{
			noun: "versions", verb: "publish", summary: "Publish a version, making it live", level: levelContainer, idFlag: "version-id", writes: true, confirm: gtm.OpPublishVersion,
			run: func(ctx context.Context, c *gtm.Client, in *invocation) (any, error) {
				version, err := c.PublishVersion(ctx, in.account, in.container, in.id)
				if err != nil {
					return nil, err
				}
				return gtm.PublishVersionOutput{Success: true, Version: *version, Message: fmt.Sprintf("Version %s is now LIVE", version.VersionID)}, nil
			},
		},
	}
}
