/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package consolemcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/snowplow/snowplow-cli/internal/util"
)

const mcpPath = "/api/agent/mcp"

func URL(consoleHost string) string {
	return strings.TrimSuffix(consoleHost, "/") + mcpPath
}

func AddTools(ctx context.Context, s *server.MCPServer, url, orgID, apiKeyID, apiKey string) (*client.Client, int, error) {
	c, err := client.NewStreamableHttpClient(url, transport.WithHTTPHeaders(map[string]string{
		"X-API-Key-ID": apiKeyID,
		"X-API-Key":    apiKey,
		"X-Org-Id":     orgID,
	}))
	if err != nil {
		return nil, 0, err
	}

	if err := c.Start(ctx); err != nil {
		_ = c.Close()
		return nil, 0, err
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "snowplow-cli", Version: util.VersionInfo}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		_ = c.Close()
		return nil, 0, fmt.Errorf("initialize: %w", err)
	}

	added := 0
	req := mcp.ListToolsRequest{}
	for {
		list, err := c.ListTools(ctx, req)
		if err != nil {
			_ = c.Close()
			return nil, 0, fmt.Errorf("list tools: %w", err)
		}
		for _, tool := range list.Tools {
			if s.GetTool(tool.Name) != nil {
				slog.Warn("skipping console tool that clashes with a local tool", "tool", tool.Name)
				continue
			}
			s.AddTool(tool, forward(c))
			added++
		}
		if list.NextCursor == "" {
			break
		}
		req.Params.Cursor = list.NextCursor
	}

	return c, added, nil
}

func forward(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.CallTool(ctx, req)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Snowplow Console MCP call failed: %v", err)), nil
		}
		return res, nil
	}
}
