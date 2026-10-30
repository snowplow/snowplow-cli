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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func fakeConsoleMCP(t *testing.T) *httptest.Server {
	t.Helper()

	// One tool per page, so AddTools has to follow the cursor.
	hosted := server.NewMCPServer("console", "test", server.WithToolCapabilities(true), server.WithPaginationLimit(1))
	hosted.AddTool(
		mcp.NewTool("list_signals_environments", mcp.WithString("filter")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("envs:" + req.GetString("filter", "")), nil
		},
	)
	hosted.AddTool(
		mcp.NewTool("get_uuid"),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("hosted"), nil
		},
	)

	h := server.NewStreamableHTTPServer(hosted, server.WithStateLess(true))
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key-ID") != "kid" || r.Header.Get("X-API-Key") != "key" || r.Header.Get("X-Org-Id") != "org" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	}))
}

func TestAddToolsForwardsCalls(t *testing.T) {
	ts := fakeConsoleMCP(t)
	defer ts.Close()

	local := server.NewMCPServer("local", "test", server.WithToolCapabilities(true))
	local.AddTool(mcp.NewTool("get_uuid"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("local"), nil
	})

	ctx := context.Background()
	c, n, err := AddTools(ctx, local, ts.URL, "org", "kid", "key")
	if err != nil {
		t.Fatalf("AddTools: %v", err)
	}
	defer func() { _ = c.Close() }()

	// get_uuid clashes with the local tool, so only one hosted tool is added.
	if n != 1 {
		t.Fatalf("expected 1 hosted tool added, got %d", n)
	}

	tool := local.GetTool("list_signals_environments")
	if tool == nil {
		t.Fatal("hosted tool not registered")
	}
	if _, ok := tool.Tool.InputSchema.Properties["filter"]; !ok {
		t.Fatalf("input schema not carried over: %+v", tool.Tool.InputSchema)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = "list_signals_environments"
	req.Params.Arguments = map[string]any{"filter": "ready"}
	res, err := tool.Handler(ctx, req)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if got := res.Content[0].(mcp.TextContent).Text; got != "envs:ready" {
		t.Fatalf("unexpected result %q", got)
	}

	res, err = local.GetTool("get_uuid").Handler(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if got := res.Content[0].(mcp.TextContent).Text; got != "local" {
		t.Fatalf("local tool was replaced, got %q", got)
	}
}

func TestAddToolsBadKey(t *testing.T) {
	ts := fakeConsoleMCP(t)
	defer ts.Close()

	local := server.NewMCPServer("local", "test")
	if _, _, err := AddTools(context.Background(), local, ts.URL, "org", "kid", "wrong"); err == nil {
		t.Fatal("expected an error for a rejected key")
	}
	if len(local.ListTools()) != 0 {
		t.Fatal("no tools should be registered when the key is rejected")
	}
}

func TestURL(t *testing.T) {
	if got := URL("https://console.snowplowanalytics.com/"); got != "https://console.snowplowanalytics.com/api/agent/mcp" {
		t.Fatalf("unexpected url %q", got)
	}
}
