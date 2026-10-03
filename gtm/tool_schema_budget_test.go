package gtm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Keep the default tool surface below the threshold where the parity roadmap
// requires configurable tool groups. Run cmd/tool-schema-report for details.
func TestToolSchemaBudget(t *testing.T) {
	const maxBytes = 80_000

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := mcp.NewServer(&mcp.Implementation{Name: "schema-test", Version: "1"}, nil)
	RegisterTools(server)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "schema-test", Version: "1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > maxBytes {
		t.Fatalf("tools/list result is %d bytes, above %d; run go run ./cmd/tool-schema-report and introduce configurable tool groups", len(payload), maxBytes)
	}
}
