package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// config is the optional file holding default IDs.
type config struct {
	AccountID   string `json:"accountId"`
	ContainerID string `json:"containerId"`
	WorkspaceID string `json:"workspaceId"`
}

// idRe matches GTM IDs. They are placed in URL paths, so nothing else is
// accepted: a slash could address a different resource.
var idRe = regexp.MustCompile(`^[0-9]+$`)

// resolveIDs fills the IDs a command needs from flags, then the environment,
// then the config file, and validates them. Only the user's own config is read,
// never a file in the working directory, so a checked-out repository cannot
// redirect writes.
func (a *App) resolveIDs(cmd *command, opts *options) error {
	ids := []struct {
		level      level
		flag, env  string
		value      *string
		fromConfig func(config) string
	}{
		{levelAccount, "account", "GTM_ACCOUNT_ID", &opts.account, func(c config) string { return c.AccountID }},
		{levelContainer, "container", "GTM_CONTAINER_ID", &opts.container, func(c config) string { return c.ContainerID }},
		{levelWorkspace, "workspace", "GTM_WORKSPACE_ID", &opts.workspace, func(c config) string { return c.WorkspaceID }},
	}

	var file *config
	for _, id := range ids {
		if cmd.level < id.level {
			continue
		}
		if *id.value == "" {
			*id.value = a.Getenv(id.env)
		}
		if *id.value == "" {
			if file == nil {
				loaded, err := a.loadConfig()
				if err != nil {
					return err
				}
				file = &loaded
			}
			*id.value = id.fromConfig(*file)
		}
		if *id.value == "" {
			return usageError("--%s is required (or set %s, or add it to the config file)", id.flag, id.env)
		}
		if !idRe.MatchString(*id.value) {
			return usageError("--%s must be a numeric ID, got %q", id.flag, *id.value)
		}
	}

	if cmd.idFlag != "" {
		if opts.id == "" {
			return usageError("--%s is required", cmd.idFlag)
		}
		if !idRe.MatchString(opts.id) {
			return usageError("--%s must be a numeric ID, got %q", cmd.idFlag, opts.id)
		}
	}
	if cmd.takesFile && opts.file == "" {
		return usageError("--file is required")
	}
	return nil
}

func (a *App) loadConfig() (config, error) {
	var c config
	path := a.Getenv("GTM_CONFIG")
	explicit := path != ""
	if !explicit {
		dir := a.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			if home := a.Getenv("HOME"); home != "" {
				dir = filepath.Join(home, ".config")
			} else {
				dir = a.Getenv("APPDATA") // Windows
			}
		}
		// A relative path would resolve against the working directory.
		if !filepath.IsAbs(dir) {
			return c, nil
		}
		path = filepath.Join(dir, "gtm", "config.json")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && !explicit {
		return c, nil
	}
	if err != nil {
		return c, usageError("cannot read config file: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return c, usageError("invalid config file %s: %v", path, err)
	}
	return c, nil
}
