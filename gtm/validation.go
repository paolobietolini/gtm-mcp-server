package gtm

import (
	"fmt"
	"strings"
)

// ValidateTagInput validates tag creation/update inputs.
func ValidateTagInput(name, tagType string, firingTriggerIDs []string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("tag name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("tag name must be 256 characters or less")
	}
	if strings.TrimSpace(tagType) == "" {
		return fmt.Errorf("tag type is required")
	}
	if len(firingTriggerIDs) == 0 {
		return fmt.Errorf("at least one firing trigger ID is required")
	}
	for _, id := range firingTriggerIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("firing trigger ID cannot be empty")
		}
	}
	return nil
}

// ValidateTriggerInput validates trigger creation inputs.
func ValidateTriggerInput(name, triggerType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("trigger name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("trigger name must be 256 characters or less")
	}
	if strings.TrimSpace(triggerType) == "" {
		return fmt.Errorf("trigger type is required")
	}
	return nil
}

// ValidateVariableInput validates variable creation inputs.
func ValidateVariableInput(name, varType string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("variable name is required")
	}
	if len(name) > 256 {
		return fmt.Errorf("variable name must be 256 characters or less")
	}
	if strings.TrimSpace(varType) == "" {
		return fmt.Errorf("variable type is required")
	}
	return nil
}

// ValidateWorkspacePath validates workspace path components.
func ValidateWorkspacePath(accountID, containerID, workspaceID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("account ID is required")
	}
	if strings.TrimSpace(containerID) == "" {
		return fmt.Errorf("container ID is required")
	}
	if strings.TrimSpace(workspaceID) == "" {
		return fmt.Errorf("workspace ID is required")
	}
	return nil
}

// BuildWorkspacePath constructs a workspace path from IDs.
func BuildWorkspacePath(accountID, containerID, workspaceID string) string {
	return fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s",
		accountID, containerID, workspaceID)
}
