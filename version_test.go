package main

import "testing"

func TestServerVersionMetadata(t *testing.T) {
	for _, tt := range []struct {
		name, data, want string
		invalid          bool
	}{
		{"version", `{"version":"2.3.4","name":"example/server"}`, "2.3.4", false},
		{"prerelease", `{"version":"2.3.4-rc.1"}`, "2.3.4-rc.1", false},
		{"malformed", `{`, "", true},
		{"missing", `{}`, "", true},
		{"empty", `{"version":""}`, "", true},
		{"whitespace", `{"version":"  "}`, "", true},
		{"wrong type", `{"version":123}`, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServerVersion([]byte(tt.data))
			if (err != nil) != tt.invalid || got != tt.want {
				t.Fatalf("version=%q err=%v", got, err)
			}
		})
	}
}

func TestEmbeddedServerVersion(t *testing.T) {
	// The production metadata must also pass validation; no duplicated version
	// literal here to become stale at the next release.
	if _, err := parseServerVersion(serverMetadata); err != nil {
		t.Fatal(err)
	}
}
