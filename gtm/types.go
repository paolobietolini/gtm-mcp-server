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
	Negate    bool        `json:"negate,omitempty"`
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

// BuiltInVariable represents an enabled built-in variable in a workspace.
type BuiltInVariable struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// ClientInfo represents a GTM client (server-side containers only).
type ClientInfo struct {
	ClientID       string `json:"clientId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Priority       int64  `json:"priority,omitempty"`
	// Using any to avoid recursive type cycle in schema generation.
	Parameter      any    `json:"parameter,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Path           string `json:"path"`
	Fingerprint    string `json:"fingerprint"`
}

// ClientInput represents input for creating/updating a client.
type ClientInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Priority  int64       `json:"priority,omitempty"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

// CreatedClient represents the result of creating a client.
type CreatedClient struct {
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// TransformationInfo represents a GTM transformation (server-side containers only).
type TransformationInfo struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	// Using any to avoid recursive type cycle in schema generation.
	Parameter      any    `json:"parameter,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
	Path           string `json:"path"`
	Fingerprint    string `json:"fingerprint"`
}

// TransformationInput represents input for creating/updating a transformation.
type TransformationInput struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Parameter []Parameter `json:"parameter,omitempty"`
	Notes     string      `json:"notes,omitempty"`
}

// CreatedTransformation represents the result of creating a transformation.
type CreatedTransformation struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint"`
}
