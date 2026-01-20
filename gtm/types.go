package gtm

// Parameter represents a GTM parameter structure.
// Used in tags, triggers, and variables.
type Parameter struct {
	Type  string      `json:"type"`            // "template", "boolean", "integer", "list", "map"
	Key   string      `json:"key"`
	Value string      `json:"value,omitempty"`
	List  []Parameter `json:"list,omitempty"`
	Map   []Parameter `json:"map,omitempty"`
}

// TagInput represents input for creating/updating a tag.
type TagInput struct {
	Name               string      `json:"name"`
	Type               string      `json:"type"`
	FiringTriggerId    []string    `json:"firingTriggerId"`
	BlockingTriggerId  []string    `json:"blockingTriggerId,omitempty"`
	Parameter          []Parameter `json:"parameter,omitempty"`
	Notes              string      `json:"notes,omitempty"`
	Paused             bool        `json:"paused,omitempty"`
	TagFiringOption    string      `json:"tagFiringOption,omitempty"`
}

// TriggerInput represents input for creating/updating a trigger.
type TriggerInput struct {
	Name              string      `json:"name"`
	Type              string      `json:"type"`
	Filter            []Condition `json:"filter,omitempty"`
	AutoEventFilter   []Condition `json:"autoEventFilter,omitempty"`
	CustomEventFilter []Condition `json:"customEventFilter,omitempty"`
	EventName         *Parameter  `json:"eventName,omitempty"`
	Parameter         []Parameter `json:"parameter,omitempty"` // For trigger groups: member trigger references
	Notes             string      `json:"notes,omitempty"`
}

// Condition represents a filter condition for triggers.
type Condition struct {
	Type      string      `json:"type"` // "equals", "contains", "startsWith", etc.
	Parameter []Parameter `json:"parameter"`
}

// VariableInput represents input for creating a variable.
type VariableInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

// VersionInput represents input for creating a version.
type VersionInput struct {
	Name  string `json:"name,omitempty"`
	Notes string `json:"notes,omitempty"`
}

// CreatedTag represents the result of creating a tag.
type CreatedTag struct {
	TagID       string `json:"tagId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedTrigger represents the result of creating a trigger.
type CreatedTrigger struct {
	TriggerID   string `json:"triggerId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedVariable represents the result of creating a variable.
type CreatedVariable struct {
	VariableID  string `json:"variableId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// CreatedVersion represents the result of creating a version.
type CreatedVersion struct {
	VersionID string `json:"containerVersionId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}
