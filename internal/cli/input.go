package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/paolobietolini/gtm-mcp-server/gtm"
)

// decodeStrict decodes exactly one JSON object into target, rejecting unknown
// fields so that a misspelled key cannot be silently ignored. It returns the
// lower-cased names of the keys present with a non-null value; the decoder
// matches keys without regard to case, so presence must do the same.
func decodeStrict(data []byte, target any) (map[string]bool, error) {
	// Unmarshal also rejects anything following the first value.
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil && !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	if keys == nil {
		return nil, errors.New("input must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	present := make(map[string]bool, len(keys))
	for key, value := range keys {
		if string(bytes.TrimSpace(value)) != "null" {
			present[strings.ToLower(key)] = true
		}
	}
	return present, nil
}

// decodeTagInput reads a tag file. For updates, a key that is absent or null
// keeps the current value.
func decodeTagInput(data []byte) (*gtm.TagInput, error) {
	input := &gtm.TagInput{}
	present, err := decodeStrict(data, input)
	if err != nil {
		return nil, err
	}
	input.HasParameter = present["parameter"]
	input.HasNotes = present["notes"]
	input.HasPaused = present["paused"]
	input.HasSetupTag = present["setuptag"]
	input.ClearSetupTag = input.HasSetupTag && len(input.SetupTag) == 0
	input.HasTeardownTag = present["teardowntag"]
	input.ClearTeardownTag = input.HasTeardownTag && len(input.TeardownTag) == 0
	input.HasConsentSettings = input.ConsentStatus != ""
	if len(input.ConsentTypes) > 0 && input.ConsentStatus == "" {
		return nil, errors.New("consentTypes requires consentStatus")
	}
	return input, nil
}

func decodeTriggerInput(data []byte) (*gtm.TriggerInput, error) {
	input := &gtm.TriggerInput{}
	present, err := decodeStrict(data, input)
	if err != nil {
		return nil, err
	}
	input.HasNotes = present["notes"]
	return input, nil
}

func decodeVariableInput(data []byte) (*gtm.VariableInput, error) {
	input := &gtm.VariableInput{}
	present, err := decodeStrict(data, input)
	if err != nil {
		return nil, err
	}
	input.HasParameter = present["parameter"]
	input.HasNotes = present["notes"]
	return input, nil
}
