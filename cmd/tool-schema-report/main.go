// Command tool-schema-report measures the MCP tool definitions sent to clients.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gtm-mcp-server/gtm"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolSize struct {
	name              string
	bytes             int
	inputSchemaBytes  int
	outputSchemaBytes int
	descriptionBytes  int
}

func main() {
	top := flag.Int("top", 10, "number of largest tool definitions to print")
	groupList := flag.String("groups", "", "comma-separated GTM tool groups; empty uses the default set")
	flag.Parse()
	if *top < 0 {
		log.Fatal("-top must be zero or greater")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tool-schema-report",
		Version: "development",
	}, nil)
	var names []string
	if strings.TrimSpace(*groupList) != "" {
		names = strings.Split(*groupList, ",")
	}
	groups, err := gtm.ParseToolGroups(names)
	if err != nil {
		log.Fatal(err)
	}
	gtm.RegisterToolsForGroups(server, groups)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		log.Fatalf("connect server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "tool-schema-report",
		Version: "development",
	}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("list tools: %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("marshal tools/list result: %v", err)
	}

	sizes := make([]toolSize, 0, len(result.Tools))
	for _, tool := range result.Tools {
		definition, err := json.Marshal(tool)
		if err != nil {
			log.Fatalf("marshal tool %q: %v", tool.Name, err)
		}
		inputSchema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			log.Fatalf("marshal input schema for %q: %v", tool.Name, err)
		}
		outputSchema, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			log.Fatalf("marshal output schema for %q: %v", tool.Name, err)
		}
		sizes = append(sizes, toolSize{
			name:              tool.Name,
			bytes:             len(definition),
			inputSchemaBytes:  len(inputSchema),
			outputSchemaBytes: len(outputSchema),
			descriptionBytes:  len(tool.Description),
		})
	}
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].bytes == sizes[j].bytes {
			return sizes[i].name < sizes[j].name
		}
		return sizes[i].bytes > sizes[j].bytes
	})

	fmt.Printf("groups: %s\n", strings.Join(groups.Names(), ","))
	fmt.Printf("tools: %d\n", len(result.Tools))
	fmt.Printf("tools/list result bytes: %d\n", len(payload))
	fmt.Printf("estimated tokens at 4 bytes/token: %d\n", (len(payload)+3)/4)
	fmt.Println("largest tool definitions (total, input schema, output schema, description):")
	for i, size := range sizes {
		if i >= *top {
			break
		}
		fmt.Printf("%d\t%d\t%d\t%d\t%s\n",
			size.bytes,
			size.inputSchemaBytes,
			size.outputSchemaBytes,
			size.descriptionBytes,
			size.name,
		)
	}
}
