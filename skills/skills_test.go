// Package skills_test validates the Agent Skills in this directory. The skills
// are plain files; this package contains only tests.
package skills_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"gtm-mcp-server/gtm"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var update = flag.Bool("update", false, "rewrite generated skill references from their sources")

const (
	// sharedDir holds sources used only by skills. It is not a skill.
	sharedDir     = "shared"
	serverDocsDir = "../gtm/bestpractices/docs"
)

// generated maps each source file to the skills that carry a copy of it in
// references/. Each skill folder must work on its own (zip upload, copy to a
// skills directory), so shared content is duplicated and checked here.
var generated = map[string][]string{
	serverDocsDir + "/naming-organization.md": {"gtm-container-audit", "gtm-safe-edit", "gtm-tracking-plan"},
	serverDocsDir + "/safe-edit-workflow.md":  {"gtm-safe-edit"},
	serverDocsDir + "/ga4-consent.md":         {"gtm-container-audit", "gtm-ga4-consent", "gtm-tracking-plan"},
	serverDocsDir + "/server-side.md":         {"gtm-container-audit", "gtm-server-side"},
	serverDocsDir + "/tool-input-formats.md":  {"gtm-mcp"},
	sharedDir + "/container-export.md": {
		"gtm-container-audit", "gtm-ga4-consent", "gtm-safe-edit", "gtm-server-side", "gtm-tracking-plan",
	},
}

func skillNames(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != sharedDir {
			names = append(names, e.Name())
		}
	}
	return names
}

func markdownFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Declared first so that -update writes the copies before other tests read them.
func TestGeneratedReferencesMatchSources(t *testing.T) {
	copies := map[string]bool{}
	for source, skills := range generated {
		want, err := os.ReadFile(source)
		if err != nil {
			t.Errorf("source: %v", err)
			continue
		}
		for _, skill := range skills {
			dst := filepath.Join(skill, "references", filepath.Base(source))
			copies[dst] = true
			if *update {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dst, want, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := os.ReadFile(dst)
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("%s differs from %s; edit the source, then run: go test ./skills -update", dst, source)
			}
		}
	}

	// A server doc added later must reach a skill. The index lists server
	// resource URIs and has no use outside the server.
	docs, err := filepath.Glob(serverDocsDir + "/*.md")
	if err != nil || len(docs) == 0 {
		t.Fatalf("no server docs found: %v", err)
	}
	for _, doc := range docs {
		if filepath.Base(doc) != "index.md" && len(generated[filepath.ToSlash(doc)]) == 0 {
			t.Errorf("%s is not copied into any skill; add it to generated", doc)
		}
	}

	// A hand-made file must not shadow a generated name, where it could drift.
	for source := range generated {
		matches, err := filepath.Glob(filepath.Join("*", "references", filepath.Base(source)))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range matches {
			if !copies[m] {
				t.Errorf("%s is not listed in generated", m)
			}
		}
	}
}

var (
	nameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	xmlRe  = regexp.MustCompile(`<[^>]+>`)
)

// parseFrontmatter accepts only single-line "key: value" entries with unquoted
// values that are also valid YAML. This is stricter than YAML on purpose: it
// needs no YAML dependency and every client reads it the same way.
func parseFrontmatter(t *testing.T, doc string) (fields map[string]string, body string) {
	t.Helper()
	rest, ok := strings.CutPrefix(doc, "---\n")
	if !ok {
		t.Fatal("SKILL.md must start with a --- frontmatter block")
	}
	header, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatal("frontmatter block is not closed")
	}
	fields = map[string]string{}
	for line := range strings.SplitSeq(header, "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			t.Fatalf("frontmatter line is not \"key: value\": %q", line)
		}
		if strings.Contains(value, ": ") || strings.Contains(value, " #") ||
			strings.HasSuffix(value, ":") || value == "" || strings.ContainsRune("\"'[]{}&*!|>%@`#-?, ", rune(value[0])) {
			t.Errorf("%s is not a safe unquoted YAML value: %q", key, value)
		}
		if _, dup := fields[key]; dup {
			t.Errorf("duplicate frontmatter key %q", key)
		}
		fields[key] = value
	}
	return fields, body
}

func TestSkillFrontmatter(t *testing.T) {
	names := skillNames(t)
	if len(names) == 0 {
		t.Fatal("no skills found")
	}
	// Keys accepted by claude.ai uploads, which match the Agent Skills spec.
	allowed := []string{"name", "description", "license", "compatibility", "metadata", "allowed-tools"}

	for _, dir := range names {
		t.Run(dir, func(t *testing.T) {
			fields, body := parseFrontmatter(t, read(t, filepath.Join(dir, "SKILL.md")))
			for key := range fields {
				if !slices.Contains(allowed, key) {
					t.Errorf("unsupported frontmatter key %q", key)
				}
			}
			name := fields["name"]
			if name != dir {
				t.Errorf("name %q does not match folder %q", name, dir)
			}
			if !nameRe.MatchString(name) || len(name) > 64 {
				t.Errorf("name %q must be lowercase letters, digits and single hyphens, at most 64 characters", name)
			}
			if strings.Contains(name, "claude") || strings.Contains(name, "anthropic") {
				t.Errorf("name %q uses a reserved word", name)
			}
			desc := fields["description"]
			if n := utf8.RuneCountInString(desc); n == 0 || n > 1024 {
				t.Errorf("description must be 1-1024 characters, got %d", n)
			}
			if !strings.Contains(desc, "Use when") {
				t.Errorf("description must say when to use the skill (\"Use when ...\"): %q", desc)
			}
			if xmlRe.MatchString(desc) {
				t.Errorf("description must not contain XML tags: %q", desc)
			}
			if n := strings.Count(body, "\n"); n >= 500 {
				t.Errorf("SKILL.md body has %d lines, keep it under 500", n)
			}
		})
	}
}

var (
	linkRe = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	pathRe = regexp.MustCompile("`((?:references|scripts|assets)/[^`\\s]+)`")
)

func TestSkillReferences(t *testing.T) {
	for _, dir := range skillNames(t) {
		t.Run(dir, func(t *testing.T) {
			for _, file := range markdownFiles(t, dir) {
				doc := read(t, file)
				var targets []string
				for _, m := range linkRe.FindAllStringSubmatch(doc, -1) {
					target, _, _ := strings.Cut(m[1], "#")
					if target != "" && !strings.Contains(target, "://") {
						targets = append(targets, filepath.Join(filepath.Dir(file), target))
					}
				}
				for _, m := range pathRe.FindAllStringSubmatch(doc, -1) {
					targets = append(targets, filepath.Join(dir, m[1]))
				}
				for _, target := range targets {
					if !strings.HasPrefix(target, dir+string(filepath.Separator)) {
						t.Errorf("%s references %s, outside the skill folder", file, target)
					} else if _, err := os.Stat(target); err != nil {
						t.Errorf("%s references missing file %s", file, target)
					}
				}
			}

			// References stay one level deep: SKILL.md links every file itself.
			entry := read(t, filepath.Join(dir, "SKILL.md"))
			for _, file := range markdownFiles(t, dir) {
				rel, err := filepath.Rel(dir, file)
				if err != nil {
					t.Fatal(err)
				}
				if rel != "SKILL.md" && !strings.Contains(entry, "]("+filepath.ToSlash(rel)+")") {
					t.Errorf("SKILL.md does not link %s", rel)
				}
			}
		})
	}
}

// toolRe matches backticked identifiers that look like MCP tool names.
var toolRe = regexp.MustCompile("`((?:list|get|create|update|delete|publish|enable|disable|import|move|lookup|resolve|revert|sync|quick|set|undelete|link|combine|reauthorize|bulk)_[a-z0-9_]+)`")

func TestSkillToolNamesExist(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "skills-test", Version: "test"}, nil)
	groups, err := gtm.ParseToolGroups([]string{"all"})
	if err != nil {
		t.Fatal(err)
	}
	gtm.RegisterToolsForGroups(server, groups)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "skills-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	known := map[string]bool{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		known[tool.Name] = true
	}

	files := markdownFiles(t, ".")
	found := 0
	for _, file := range files {
		for _, m := range toolRe.FindAllStringSubmatch(read(t, file), -1) {
			found++
			if !known[m[1]] {
				t.Errorf("%s names unknown MCP tool %q", file, m[1])
			}
		}
	}
	if found == 0 {
		t.Error("no tool names found in skills")
	}
}
