package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Embed registry metadata so standalone binaries and Docker images use the
// same version without depending on files in the runtime working directory.
//
//go:embed server.json
var serverMetadata []byte

func parseServerVersion(data []byte) (string, error) {
	var metadata struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return "", fmt.Errorf("parse server.json: %w", err)
	}
	if strings.TrimSpace(metadata.Version) == "" {
		return "", fmt.Errorf("server.json version must not be empty")
	}
	return metadata.Version, nil
}
