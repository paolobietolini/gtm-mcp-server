# Google Tag Manager MCP Server

This MCP server for Google Tag Manager aims to simplify and accelerate common operations such as tag creation, auditing, container creation, and publishing.

A complete list of available tools will be released as soon as the server is finalized.

To provide better context to your LLM and provide it with the GTM API schema, I recommend having it read the documents in this [repository](https://github.com/paolobietolini/gtm-api-for-llms) or adding the following [skill](https://github.com/paolobietolini/gtm-api-for-llms/tree/main/skills/gtm-api).

---

## Features (Roadmap)

* **Automated Audits:** Quickly scan containers for naming convention consistency and broken tags.
* **Rapid Deployment:** Create and publish tags, triggers, and variables via natural language commands.
* **Container Management:** Programmatically create and configure new containers.
* **Version Control:** List, compare, and revert to previous container versions.

## Architecture

As a remote MCP server, this tool operates over HTTP/SSE (Server-Sent Events), allowing for centralized management and access without local installation of the GTM logic.




## Status

This project was started in January 2026. [Watch](https://github.com/paolobietolini/gtm-mcp-server) to follow updates, or feel free to open [Pull Requests](https://github.com/paolobietolini/gtm-mcp-server/pulls) to contribute.

## Author

Paolo Bietolini

<mcp at paolobietolini dot com>

## License

[BSD-3](https://github.com/paolobietolini/gtm-mcp-server/LICENSE)
