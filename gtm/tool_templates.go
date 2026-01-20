package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetTagTemplatesInput struct{}
type GetTagTemplatesOutput struct {
	Templates []TagTemplate `json:"templates"`
	Usage     string        `json:"usage"`
}

func registerGetTagTemplates(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTagTemplatesInput) (*mcp.CallToolResult, GetTagTemplatesOutput, error) {
		templates := GetTagTemplates()
		return nil, GetTagTemplatesOutput{
			Templates: templates,
			Usage: `These templates show the correct parameter structure for creating GTM tags.

IMPORTANT - Common mistakes to avoid:
1. For GA4 Event tags (gaawe), use measurementIdOverride with an empty measurementId
2. Event parameters use name/value pairs in maps, NOT direct key names
3. For ecommerce, set sendEcommerceData=true and getEcommerceDataFrom=dataLayer

Copy the parameters JSON and modify values as needed when calling create_tag.`,
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_tag_templates",
		Description: "Get example parameter structures for creating GTM tags. Use this BEFORE creating GA4 or complex tags to see the correct parameter format.",
	}, handler)
}

type GetTriggerTemplatesInput struct{}
type GetTriggerTemplatesOutput struct {
	Templates []TriggerTemplate `json:"templates"`
	Usage     string            `json:"usage"`
}

func registerGetTriggerTemplates(server *mcp.Server) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input GetTriggerTemplatesInput) (*mcp.CallToolResult, GetTriggerTemplatesOutput, error) {
		templates := GetTriggerTemplates()
		return nil, GetTriggerTemplatesOutput{
			Templates: templates,
			Usage: `These templates show the correct structure for creating GTM triggers.

For customEvent triggers, use customEventFilterJson parameter.
For pageview triggers with conditions, use filterJson parameter.
For click/form triggers with conditions, use autoEventFilterJson parameter.`,
		}, nil
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_trigger_templates",
		Description: "Get example structures for creating GTM triggers. Use this to see the correct format for different trigger types.",
	}, handler)
}
