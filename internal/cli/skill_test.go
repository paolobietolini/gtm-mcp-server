package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// skillCommandRe matches "gtm-cli <noun> <verb>" in backticks or at the start of a
// shell line.
var skillCommandRe = regexp.MustCompile("(?m)(?:`|^)gtm-cli ([a-z]+) ([a-z]+)")

// Skills teach agents to run commands, so every command they name must exist.
func TestSkillsNameRealCommands(t *testing.T) {
	files, err := filepath.Glob("../../skills/*/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range skillCommandRe.FindAllStringSubmatch(string(data), -1) {
			named[m[1]+" "+m[2]] = true
			if findCommand(m[1], m[2]) == nil {
				t.Errorf("%s names unknown command %q", file, m[0])
			}
		}
	}
	// The CLI skill is the 1:1 reference, so it must cover every command.
	data, err := os.ReadFile("../../skills/gtm-cli/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	covered := map[string]bool{}
	for _, m := range skillCommandRe.FindAllStringSubmatch(string(data), -1) {
		covered[m[1]+" "+m[2]] = true
	}
	for _, cmd := range commands {
		if !covered[cmd.noun+" "+cmd.verb] {
			t.Errorf("skills/gtm-cli/SKILL.md does not show `gtm-cli %s %s`", cmd.noun, cmd.verb)
		}
	}
}
