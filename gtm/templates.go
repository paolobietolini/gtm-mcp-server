package gtm

// TagTemplate provides example parameter structures for creating tags.
type TagTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Parameters  string `json:"parameters"`
	Notes       string `json:"notes"`
}

// GetTagTemplates returns example parameter structures for common tag types.
// These templates help LLMs create tags with the correct GTM API parameter format.
func GetTagTemplates() []TagTemplate {
	return []TagTemplate{
		{
			Name:        "GA4 Configuration",
			Description: "Google Analytics 4 configuration tag (fires on all pages)",
			Type:        "gaawc",
			Parameters: `[
  {"type": "template", "key": "measurementId", "value": "G-XXXXXXXXXX"}
]`,
			Notes: "Use gaawc type for GA4 Config tags. The measurementId should be your GA4 Measurement ID.",
		},
		{
			Name:        "GA4 Event (Simple)",
			Description: "Google Analytics 4 event tag with custom event name",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "custom_event_name"}
]`,
			Notes: "Use gaawe type for GA4 Event tags. measurementId must be empty tagReference, use measurementIdOverride for the actual value (variable reference or literal).",
		},
		{
			Name:        "GA4 Event with Parameters",
			Description: "Google Analytics 4 event tag with custom parameters",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "button_click"},
  {"type": "list", "key": "eventParameters", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "button_id"},
      {"type": "template", "key": "value", "value": "{{Click ID}}"}
    ]},
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "button_text"},
      {"type": "template", "key": "value", "value": "{{Click Text}}"}
    ]}
  ]}
]`,
			Notes: "Event parameters use name/value pairs inside map structures. Do NOT use the parameter name as the key directly.",
		},
		{
			Name:        "GA4 Ecommerce Purchase",
			Description: "Google Analytics 4 ecommerce purchase event (reads items from dataLayer)",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "purchase"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"},
  {"type": "list", "key": "eventParameters", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "transaction_id"},
      {"type": "template", "key": "value", "value": "{{DL - Transaction ID}}"}
    ]}
  ]}
]`,
			Notes: "For ecommerce events, set sendEcommerceData=true and getEcommerceDataFrom=dataLayer. The items array will be read automatically from the dataLayer ecommerce object.",
		},
		{
			Name:        "GA4 Ecommerce Add to Cart",
			Description: "Google Analytics 4 ecommerce add_to_cart event",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "add_to_cart"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"}
]`,
			Notes: "Similar to purchase, but for add_to_cart event. Items are read from dataLayer.",
		},
		{
			Name:        "GA4 Ecommerce View Item",
			Description: "Google Analytics 4 ecommerce view_item event",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "view_item"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"}
]`,
			Notes: "For product detail page views. Items are read from dataLayer.",
		},
		{
			Name:        "Custom HTML",
			Description: "Custom HTML tag for arbitrary JavaScript",
			Type:        "html",
			Parameters: `[
  {"type": "template", "key": "html", "value": "<script>\n  console.log('Hello from GTM!');\n</script>"}
]`,
			Notes: "Use html type for custom JavaScript. The html parameter contains the script.",
		},
		{
			Name:        "Custom Image (Pixel)",
			Description: "Custom image tag for tracking pixels",
			Type:        "img",
			Parameters: `[
  {"type": "template", "key": "url", "value": "https://example.com/pixel.gif?event=pageview"},
  {"type": "boolean", "key": "useCacheBuster", "value": "true"},
  {"type": "template", "key": "cacheBusterQueryParam", "value": "gtmcb"}
]`,
			Notes: "Use img type for tracking pixels. Enable cacheBuster to prevent caching.",
		},
	}
}

// TriggerTemplate provides example structures for creating triggers.
type TriggerTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	FilterJSON  string `json:"filterJson,omitempty"`
	Notes       string `json:"notes"`
}

// GetTriggerTemplates returns example structures for common trigger types.
func GetTriggerTemplates() []TriggerTemplate {
	return []TriggerTemplate{
		{
			Name:        "All Pages",
			Description: "Fires on every page view",
			Type:        "pageview",
			Notes:       "Simple pageview trigger with no filters.",
		},
		{
			Name:        "Specific Page",
			Description: "Fires on a specific page URL",
			Type:        "pageview",
			FilterJSON: `[
  {"type": "contains", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{Page URL}}"},
    {"type": "template", "key": "arg1", "value": "/checkout"}
  ]}
]`,
			Notes: "Use filterJson to match specific pages. arg0 is the variable, arg1 is the value to match.",
		},
		{
			Name:        "Custom Event",
			Description: "Fires on a dataLayer custom event",
			Type:        "customEvent",
			FilterJSON: `[
  {"type": "equals", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{_event}}"},
    {"type": "template", "key": "arg1", "value": "purchase"}
  ]}
]`,
			Notes: "For customEvent triggers, use customEventFilterJson with the event name. The {{_event}} variable matches the dataLayer event name.",
		},
		{
			Name:        "Click - All Elements",
			Description: "Fires on all element clicks",
			Type:        "linkClick",
			Notes:       "Use linkClick for click triggers. Add autoEventFilterJson to filter by click element properties.",
		},
		{
			Name:        "Form Submission",
			Description: "Fires on form submissions",
			Type:        "formSubmission",
			Notes:       "Use formSubmission type. Add autoEventFilterJson to filter by form properties.",
		},
	}
}
