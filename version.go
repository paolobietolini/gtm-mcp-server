package main

import (
	_ "embed"
	"fmt"
	"strings"
)

// The release version lives in the VERSION file at the repository root. It is
// embedded so standalone binaries and Docker images report the same version
// without depending on files in the runtime working directory, and the release
// workflow rejects a tag that does not match it.
//
//go:embed VERSION
var versionFile string

func parseServerVersion(data string) (string, error) {
	version := strings.TrimSpace(data)
	if version == "" {
		return "", fmt.Errorf("VERSION must not be empty")
	}
	if strings.ContainsAny(version, " \t\r\n") {
		return "", fmt.Errorf("VERSION must contain a single version string, got %q", version)
	}
	return version, nil
}
