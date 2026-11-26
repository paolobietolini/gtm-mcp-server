package gtm

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The MCP tools must keep answering an unconfirmed call with success: false and
// the shared message, before any credentials are needed.
func TestMCPToolsUseSharedConfirmGuard(t *testing.T) {
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "guard-test", Version: "test"}, nil)
	RegisterTools(server)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "guard-test", Version: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	ws := map[string]any{"accountId": "1", "containerId": "2", "workspaceId": "3", "confirm": false}
	with := func(key string) map[string]any {
		args := map[string]any{key: "4"}
		for k, v := range ws {
			args[k] = v
		}
		return args
	}
	for op, args := range map[Operation]map[string]any{
		OpDeleteTag:      with("tagId"),
		OpDeleteTrigger:  with("triggerId"),
		OpDeleteVariable: with("variableId"),
		OpPublishVersion: {"accountId": "1", "containerId": "2", "versionId": "4", "confirm": false},
	} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: string(op), Arguments: args})
		if err != nil || result.IsError {
			t.Fatalf("%s: result=%+v err=%v", op, result, err)
		}
		out, ok := result.StructuredContent.(map[string]any)
		if !ok || out["success"] != false || out["message"] != CheckConfirm(op, false).Error() {
			t.Errorf("%s: %+v", op, result.StructuredContent)
		}
	}
}
