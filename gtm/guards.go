package gtm

import (
	"context"
	"errors"
	"fmt"
)

// Safety rules shared by the MCP tools and the CLI. A rule that protects a
// container belongs here, so that no surface can apply a weaker version.

// Operation names an operation that needs explicit confirmation. The values are
// the MCP tool names.
type Operation string

const (
	OpDeleteTag      Operation = "delete_tag"
	OpDeleteTrigger  Operation = "delete_trigger"
	OpDeleteVariable Operation = "delete_variable"
	OpPublishVersion Operation = "publish_version"
)

// ErrConfirmationRequired matches the error returned by CheckConfirm.
var ErrConfirmationRequired = errors.New("confirmation required")

type confirmationError struct{ message string }

func (e confirmationError) Error() string        { return e.message }
func (e confirmationError) Is(target error) bool { return target == ErrConfirmationRequired }

// CheckConfirm rejects an operation that was not explicitly confirmed. The
// message uses the MCP wording, "confirm: true".
func CheckConfirm(op Operation, confirm bool) error {
	if confirm {
		return nil
	}
	switch op {
	case OpDeleteTag, OpDeleteTrigger, OpDeleteVariable:
		return confirmationError{"Deletion requires confirm: true. This is a safety guard to prevent accidental deletions."}
	case OpPublishVersion:
		return confirmationError{"Publishing requires confirm: true. WARNING: This will make the version live on your website."}
	default:
		return confirmationError{string(op) + " requires confirm: true."}
	}
}

// VersionBlockedError reports why a workspace cannot be versioned.
type VersionBlockedError struct{ reason string }

func (e *VersionBlockedError) Error() string { return e.reason }

// CreateVersionFromWorkspace creates a version after checking that the
// workspace has changes and no merge conflicts.
func (c *Client) CreateVersionFromWorkspace(ctx context.Context, accountID, containerID, workspaceID string, input *VersionInput) (*CreatedVersion, error) {
	status, err := c.GetWorkspaceStatus(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, err
	}
	if !status.HasChanges {
		return nil, &VersionBlockedError{"No changes in workspace to create version from"}
	}
	if status.HasConflicts {
		return nil, &VersionBlockedError{fmt.Sprintf("Workspace has %d conflicts that must be resolved before creating a version", status.ConflictCount)}
	}
	return c.CreateVersion(ctx, accountID, containerID, workspaceID, input)
}

// NormalizeTriggerUpdate moves autoEventFilter conditions into filter for click
// and form triggers, because the GTM API silently drops them otherwise (issue
// #39). It returns a warning when it changed the input.
func NormalizeTriggerUpdate(input *TriggerInput) (warning string) {
	if len(input.AutoEventFilter) == 0 || !isClickLinkFormTrigger(input.Type) {
		return ""
	}
	input.Filter = append(input.Filter, input.AutoEventFilter...)
	input.AutoEventFilter = nil
	return "Warning: the GTM API silently ignores autoEventFilter for " +
		input.Type + " triggers (issue #39). Conditions were automatically remapped to filter."
}

// TriggerCreateMessage describes a successful trigger creation, noting the
// remapping CreateTrigger applies to click and form triggers.
func TriggerCreateMessage(input *TriggerInput) string {
	message := "Trigger created successfully"
	if isClickLinkFormTrigger(input.Type) && len(input.AutoEventFilter) > 0 {
		message += ". Note: autoEventFilter was automatically remapped to filter for " +
			input.Type + " triggers (GTM API requirement)."
	}
	return message
}
