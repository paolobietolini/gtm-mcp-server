package main

import "testing"

func TestServerVersion(t *testing.T) {
	for _, tt := range []struct {
		name, data, want string
		invalid          bool
	}{
		{"version", "2.3.4", "2.3.4", false},
		{"trailing newline", "2.3.4\n", "2.3.4", false},
		{"surrounding whitespace", "  2.3.4\r\n", "2.3.4", false},
		{"prerelease", "2.3.4-rc.1\n", "2.3.4-rc.1", false},
		{"empty", "", "", true},
		{"whitespace", "  \n", "", true},
		{"two lines", "2.3.4\n2.3.5\n", "", true},
		{"inner space", "2.3 .4", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServerVersion(tt.data)
			if (err != nil) != tt.invalid || got != tt.want {
				t.Fatalf("version=%q err=%v", got, err)
			}
		})
	}
}

func TestEmbeddedServerVersion(t *testing.T) {
	// The production VERSION file must also pass validation; no duplicated
	// version literal here to become stale at the next release.
	if _, err := parseServerVersion(versionFile); err != nil {
		t.Fatal(err)
	}
}
