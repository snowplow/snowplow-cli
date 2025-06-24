/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/snowplow/snowplow-cli/internal/config"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/snowplow/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const contextTemplate = `
# Snowplow tracking design context

data_products:
  core_value: "Every tracking implementation needs documentation - data products ARE that documentation"
  immediate_benefits:
    - "Prevents tracking drift and inconsistencies"
    - "Essential for team knowledge transfer"
    - "Eliminates 'what does this event mean?' questions"
	complete_documentation_principle: |
    Data products must document ALL events that are part of a business process or user journey, 
    including both standard Snowplow events AND custom events. This ensures:
    - Complete visibility into the data being collected
    - Proper governance and ownership of all events
    - Clear implementation guidance for developers
    - Ability to enhance standard events with custom context entities

standard_events_check:
  before_custom_implementation:
    prompt: "Have you reviewed Snowplow's standard events to see if they meet your needs?"
    reference: "https://docs.snowplow.io/docs/events/ootb-data/"
    common_categories:
			- "Application error events"
			- "App & tracker information" 
			- "App performance"
			- "Consent events (Enhanced Consent)"
			- "Device and browser information"
			- "Ecommerce events"
			- "Form tracking events"
			- "Geolocation"
			- "Media playback events"
			- "Page and screen engagement"
			- "Page and screen views"
			- "User and session identification"
    guidance: "Standard events provide for many common analytics needs. They often come with tracking implementations already provided."

event_completeness_checklist: |
  Before finalizing any data product, verify:
  - All relevant standard events are documented as event specifications
  - All custom events are documented with their schemas
  - Standard events include custom context entities where valuable
  - Event descriptions clearly indicate automatic vs manual implementation
  - Triggers show exactly when and where events fire
  - Complete user journey is covered from first interaction to conversion
  
  Common missed standard events:
  - page_view (if tracking web journeys)
  - focus_form, change_form, submit_form (if tracking forms)
  - screen_view (if tracking mobile apps)
  - link_click (if tracking navigation)

canonical_property_shadowing_policy: |
  Do NOT define custom fields in data structures that duplicate or conflict with Snowplow’s canonical (out-of-the-box) event properties
  (e.g., collector_tstamp, user_id, app_id, page_url, etc.).
  Exception:
    - The provided 'page_view' and 'page_ping' schemas are intended for use in event specifications.
    - These schemas are derived from canonical events and should be referenced in event specifications as needed.
    - Do NOT redefine or duplicate these schema as custom data structures.
  Rationale:
    Shadowing canonical properties can cause confusion, data quality issues, and make downstream analytics more difficult.
    The exception for 'page_view' and 'page_ping' ensures event specifications can use standard event types for consistency.

implementation_pattern_emphasis: |
  Standard Pattern for Data Products:
  1. Start with relevant standard events that auto-fire
  2. Add custom context entities to enhance standard events  
  3. Add custom events for business-specific requirements
  4. Document ALL events together in the data product
  5. Clearly mark which require manual implementation vs automatic

role_clarification: |
  Primary role: Tracking plan architect and data structure designer
  Secondary role: Implementation guidance (only when explicitly requested)
  
  Focus areas:
  1. What data to track (event/entity design)
  2. How to organize tracking (data products, source apps)
  3. Standard vs custom tracking decisions
  4. Business requirements analysis

  Avoid unless requested:
  1. JavaScript implementation code
  2. Tracker configuration details
  3. Deployment instructions
  4. Technical integration steps

documentation_policy: |
  Do NOT create README.md, documentation files, or implementation guides unless:
  1. The user explicitly requests documentation/guides/readme files
  2. The user asks "how do I implement this?" or similar implementation questions
  3. The user specifically mentions wanting written instructions or guides
  Default behavior: Save only the functional tracking plan files (YAML schemas) and provide 
  implementation guidance in the chat response.

tracking_plan_workflow:
  management_approach: "Git-based version control with automated deployment"
  
  typical_directory_structure: |
    tracking-plan/
    ├── data-structures/
    │   ├── events/
    │   │   ├── form_interaction.yml
    │   │   ├── page_view_enhanced.yml
    │   │   └── ...
    │   └── entities/
    │       ├── form_data.yml
    │       ├── user_context.yml
    │       └── ...
    └── data-products/
      	├── marketing_forms.yml
      	├── ecommerce_funnel.yml
    		└── source-applications/
        		├── website.yml
        		├── mobile_app.yml
        		└── ...

mandatory_validation_workflow:
	data_structures: |
		CRITICAL: After ANY data structure creation or modification, you MUST:
		1. Save/write the data structure file
		2. Immediately call validate_data_structures on the file
		3. Report validation results to the user
		4. If validation fails, fix issues and re-validate
  data_products: |
    Data products contain $ref references to source applications (e.g., "$ref: ../source-applications/website.yml").
    These cross-file dependencies require BOTH files to be validated together.
    
    ALWAYS include source application files when validating data products:
    - Single data product: Include its referenced source apps
    - Multiple data products: Include ALL referenced source apps
    - Directory validation: Use directory path to automatically include related files

## Tracking plan component parts
It is VITAL you do not deviate from their structure.

{{range .Schemas}}
## {{.Title}}

{{.Description}}

` + "```json" + `
{{.Content}}
` + "```" + `

---

{{end}}
## Usage Notes

- many tracking plan components require uuid v4 indentifiers it is vital that is a real uuid, do not guess one
- **NEVER use the 'comment' field** in entities or events - use triggers or schema narrowing instead

`

type SchemaInfo struct {
	Title       string
	Description string
	Content     string
}

type ContextData struct {
	Schemas []SchemaInfo
}

var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP (Model Context Protocol) stdio server for Snowplow validation and context",
	Long: `Start an MCP (Model Context Protocol) stdio server that provides tools for:
  - Validating Snowplow files (data-structures, data-products, source-applications)
  - Retrieving the built-in schema and rules that define how Snowplow data structures, data products, and source applications should be structured`,
	Example: `
  Claude Desktop config:
  {
    "mcpServers": {
      ...
      "snowplow-cli": {
        "command": "snowplow-cli", "args": ["mcp"]
      }
    }
  }

  VS Code '<workspace>/.vscode/mcp.json':
  {
    "servers": {
      ...
      "snowplow-cli": {
        "type": "stdio",
        "command": "snowplow-cli", "args": ["mcp"]
      }
    }
  }

  Cursor '<workspace>/.cursor/mcp.json':
  {
    "mcpServers": {
      ...
      "snowplow-cli": {
        "command": "snowplow-cli", "args": ["mcp", "--base-directory", "."]
      }
    }
  }

Note:
  This server's validation tools require filesystem paths to validate assets. For full
  functionality, your MCP client needs filesystem write access so created assets can be
  saved as files and then validated.

Setup options:
  - Enable filesystem access in your MCP client, or
  - Run alongside an MCP filesystem server (e.g., @modelcontextprotocol/server-filesystem)
`,
	RunE: runMCPServer,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := config.InitConsoleConfig(cmd); err != nil {
			logging.LogFatal(err)
		}

		return nil
	},
}

func init() {
	config.InitConsoleFlags(McpCmd)

	McpCmd.Flags().Bool("dump-context", false, "Dumps the result of the get_context tool to stdout and exits.")
	McpCmd.Flags().String("base-directory", "", "The base path to use for relative file lookups. Useful for clients that pass in relative file paths.")
}

func runMCPServer(cmd *cobra.Command, args []string) error {

	if dumpContext, _ := cmd.Flags().GetBool("dump-context"); dumpContext {
		out, err := getSnowplowContext(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	s := server.NewMCPServer(
		"Snowplow CLI MCP Server",
		util.VersionInfo,
		server.WithToolCapabilities(true),
	)

	validateDataStructuresTool := mcp.NewTool("validate_data_structures",
		mcp.WithDescription("**REQUIRED AFTER ANY DATA STRUCTURE WORK**: Validate Snowplow data structure files immediately after creating, editing, or modifying any data structure. This tool MUST be called as the final step of any data structure operation to ensure schema compliance and catch validation errors before deployment. Not for data products or source apps - only data structures (events/entities)"),
		mcp.WithArray("paths",
			mcp.Required(),
			mcp.Description("Paths to the files containing data structures to validate, directories will be walked"),
			mcp.Items(map[string]string{"type": "string"}),
		),
	)

	validateDataProductsTool := mcp.NewTool("validate_data_products",
		mcp.WithDescription("**REQUIRED AFTER ANY DATA PRODUCT OR SOURCE APPLICATION WORK**: Validate Snowplow data product and source application files immediately after creating, editing, or modifying any data products or source applications. This tool MUST be called as the final step of any data product or source application operation to ensure schema compliance, reference integrity, and catch validation errors before deployment. Validates data product event specifications, entity references, source application configurations, and cross-file dependencies. IMPORTANT: When validating data products, you MUST include all referenced source application files in the same validation call, as data products contain $ref links to source applications that require cross-file validation. Include both the data product files AND their corresponding source application files in the paths array."),
		mcp.WithArray("paths",
			mcp.Required(),
			mcp.Description("Paths to the files containing data products and source applications to validate, directories will be walked"),
			mcp.Items(map[string]string{"type": "string"}),
		),
	)

	getContextTool := mcp.NewTool("get_context",
		mcp.WithDescription("REQUIRED: Call this tool FIRST before any Snowplow analytics work. Retrieves complete JSON schemas, validation rules, and structural examples for Snowplow data-structures, data-products, and source-applications. This context includes required fields, format patterns (like Iglu URIs), cardinality rules, schema versioning (MODEL-REVISION-ADDITION), and real-world examples. Essential for understanding file structure, creating valid configurations, and providing accurate validation guidance. Without this context, Snowplow responses will lack the technical precision needed for implementation. MANDATORY: Before performing any Snowplow tracking, data product, or data structure task, you MUST call this tool and read its output. Do not proceed with any analysis, validation, or code changes until you have retrieved and reviewed this context"),
	)

	uuidTool := mcp.NewTool("get_uuid",
		mcp.WithDescription("Generates a valid reliable v4 uuid"),
		mcp.WithNumber("count", mcp.Description("how many to generate"), mcp.DefaultNumber(1)),
	)

	s.AddTool(validateDataStructuresTool, validateDataStructuresHandler(cmd))
	s.AddTool(validateDataProductsTool, validateDataProductsHandler(cmd))
	s.AddTool(getContextTool, getContextHandler)
	s.AddTool(uuidTool, genUuid)

	if err := server.ServeStdio(s); err != nil {
		if !errors.Is(err, context.Canceled) {
			return fmt.Errorf("MCP server error: %v", err)
		}
	}

	return nil
}

func genUuid(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	count := request.GetInt("count", 1)
	result := []string{}
	for range count {
		result = append(result, uuid.NewString())
	}
	return mcp.NewToolResultText(strings.Join(result, ", ")), nil
}

func pathsFromArguments(base string, args map[string]any) ([]string, error) {
	paths, ok := args["paths"].([]any)
	if !ok {
		return nil, errors.New("file_path must be strings")
	}

	filePaths := []string{}
	for _, p := range paths {
		if fp, ok := p.(string); ok {
			filePaths = append(filePaths, filepath.Join(base, fp))
		}
	}

	return filePaths, nil
}

func validateDataProductsHandler(cmd *cobra.Command) server.ToolHandlerFunc {

	logLevel := slog.LevelInfo

	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		logLevel = slog.LevelDebug
	}

	baseDir, _ := cmd.Flags().GetString("base-directory")

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = context.WithValue(ctx, util.MCPSourceContextKey{}, true)

		args := request.GetArguments()

		paths, err := pathsFromArguments(baseDir, args)
		if err != nil {
			return nil, errors.New("file_path must be strings")
		}

		var output strings.Builder
		logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
			Level: logLevel,
		}))

		outputCtx := logging.ContextWithLogger(ctx, logger)

		basePath, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		err = validation.ValidateDataProductsFromCmd(outputCtx, cmd, paths, basePath)
		if err != nil {
			logger.Error("validation", "error", err.Error())
		}

		return mcp.NewToolResultText(output.String()), nil

	}
}

func validateDataStructuresHandler(cmd *cobra.Command) server.ToolHandlerFunc {

	logLevel := slog.LevelInfo

	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		logLevel = slog.LevelDebug
	}

	baseDir, _ := cmd.Flags().GetString("base-directory")

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = context.WithValue(ctx, util.MCPSourceContextKey{}, true)

		args := request.GetArguments()

		paths, err := pathsFromArguments(baseDir, args)
		if err != nil {
			return nil, err
		}

		var output strings.Builder
		logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
			Level: logLevel,
		}))

		outputCtx := logging.ContextWithLogger(ctx, logger)

		err = validation.ValidateDataStructuresFromCmd(outputCtx, cmd, paths)
		if err != nil {
			logger.Error(err.Error())
		}

		return mcp.NewToolResultText(output.String()), nil
	}
}

func getSnowplowContext(_ context.Context) (string, error) {
	schemaMetadata := map[string]struct {
		title string
		desc  string
	}{
		"data-structure": {
			title: "Data Structure Schema",
			desc:  "Defines event and entity schemas for Snowplow tracking",
		},
		"data-product": {
			title: "Data Product Schema",
			desc:  "Defines what data to track and how it relates to business processes",
		},
		"source-application": {
			title: "Source Application Schema",
			desc:  "Defines applications/systems that generate Snowplow events",
		},
	}

	embeddedSchemas := validation.GetEmbeddedSchemas()

	var schemas []SchemaInfo
	for name, meta := range schemaMetadata {
		schemaContent, exists := embeddedSchemas[name]
		if !exists {
			return "", fmt.Errorf("schema %s not found in embedded schemas", name)
		}

		schemas = append(schemas, SchemaInfo{
			Title:       meta.title,
			Description: meta.desc,
			Content:     schemaContent,
		})
	}

	tmpl, err := template.New("context").Parse(contextTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, ContextData{Schemas: schemas})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	return result.String(), nil
}

func getContextHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	result, err := getSnowplowContext(ctx)

	return mcp.NewToolResultText(result), err
}
