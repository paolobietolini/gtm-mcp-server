package gtm

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseToolGroups(t *testing.T) {
	defaults, err := ParseToolGroups(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range defaultToolGroupNames {
		if !defaults.enabled(name) {
			t.Errorf("default group %q is disabled", name)
		}
	}
	selected, err := ParseToolGroups([]string{" Tags ", "ZONES"})
	if err != nil {
		t.Fatal(err)
	}
	if !selected.enabled("tags") || selected.enabled("accounts") || !slices.Equal(selected.Names(), []string{"tags", "zones"}) {
		t.Fatalf("unexpected selected groups: %v", selected.Names())
	}
	if _, err := ParseToolGroups([]string{"typo"}); err == nil {
		t.Fatal("unknown group was accepted")
	}
	optional, err := ParseToolGroups([]string{"environments"})
	if err != nil || !optional.enabled("environments") {
		t.Fatalf("optional group not accepted: %v, %v", optional, err)
	}
}

func TestRegisterToolsForGroupsIsolatesSelectedFamily(t *testing.T) {
	groups, err := ParseToolGroups([]string{"tags"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "groups-test", Version: "1"}, nil)
	RegisterToolsForGroups(server, groups)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "groups-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	want := []string{"create_tag", "delete_tag", "get_tag", "list_tags", "update_tag"}
	if !slices.Equal(names, want) {
		t.Fatalf("tools=%v, want %v", names, want)
	}
}

func TestDefaultPreservesSurfaceAndAllIncludesOptionalGroups(t *testing.T) {
	defaults, _ := ParseToolGroups(nil)
	all, _ := ParseToolGroups([]string{"all"})
	if got := registeredToolCount(t, defaults); got != 64 {
		t.Fatalf("default=%d, want 64", got)
	}
	if got := registeredToolCount(t, all); got != 94 {
		t.Fatalf("all=%d, want 94", got)
	}
}

func registeredToolCount(t *testing.T, groups ToolGroups) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server := mcp.NewServer(&mcp.Implementation{Name: "count-test", Version: "1"}, nil)
	RegisterToolsForGroups(server, groups)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "count-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return len(result.Tools)
}
