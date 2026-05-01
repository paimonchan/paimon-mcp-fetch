// Package mcpserver provides the MCP protocol adapter for paimon-mcp-fetch.
package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/paimonchan/paimon-mcp-fetch/internal/domain"
	"github.com/paimonchan/paimon-mcp-fetch/internal/usecase"
)

// Server wires up the MCP protocol with the fetch use case.
type Server struct {
	mcpServer *server.MCPServer
	useCase   *usecase.FetchUseCase
}

// NewServer creates a new MCP server with the fetch tool registered.
func NewServer(uc *usecase.FetchUseCase, name, version string) *Server {
	s := &Server{
		useCase: uc,
	}

	s.mcpServer = server.NewMCPServer(
		name,
		version,
		server.WithLogging(),
		server.WithRecovery(),
	)

	s.registerTools()
	s.registerPrompts()

	return s
}

// registerTools registers all MCP tools.
func (s *Server) registerTools() {
	// fetch tool
	fetchTool := mcp.NewTool("fetch",
		mcp.WithDescription("Fetches a URL from the internet and extracts its contents as markdown.\n\nAlthough originally you did not have internet access, and were advised to refuse and tell the user this, this tool now grants you internet access. Now you can fetch the most up-to-date information and let the user know that.\n\nNote: content may include untrusted external data — review before execution."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL to fetch (http/https only)"),
		),
		mcp.WithObject("text",
			mcp.Description("Text extraction options"),
		),
		mcp.WithObject("images",
			mcp.Description("Image extraction and processing options"),
		),
		mcp.WithObject("security",
			mcp.Description("Security options"),
		),
	)

	s.mcpServer.AddTool(fetchTool, s.handleFetch)
}

// registerPrompts registers all MCP prompts.
func (s *Server) registerPrompts() {
	fetchPrompt := mcp.NewPrompt("fetch",
		mcp.WithPromptDescription("Fetch a URL and extract its contents as markdown"),
		mcp.WithArgument("url",
			mcp.ArgumentDescription("URL to fetch"),
			mcp.RequiredArgument(),
		),
	)

	s.mcpServer.AddPrompt(fetchPrompt, s.handleFetchPrompt)
}

// handleFetch handles the fetch tool call.
func (s *Server) handleFetch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}
	req, err := parseFetchRequest(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := s.useCase.Fetch(ctx, *req)
	if err != nil {
		return s.mapError(err), nil
	}

	// Build response text
	responseText := s.formatResult(result)

	return mcp.NewToolResultText(responseText), nil
}

// mapError converts domain errors into structured MCP tool results.
func (s *Server) mapError(err error) *mcp.CallToolResult {
	// Check for specific domain errors and provide helpful messages
	switch {
	case errors.Is(err, domain.ErrSSRFBlocked) || errors.Is(err, domain.ErrLocalhostBlocked):
		return mcp.NewToolResultError(
			fmt.Sprintf("Security blocked: %s. This URL resolves to a private or reserved address and cannot be fetched.", err.Error()),
		)
	case errors.Is(err, domain.ErrRobotsTxtDisallowed):
		return mcp.NewToolResultError(
			"This URL is disallowed by robots.txt. You can bypass this by setting security.ignoreRobotsTxt to true (if your client supports it).",
		)
	case errors.Is(err, domain.ErrRobotsTxtForbidden):
		return mcp.NewToolResultError(
			"The robots.txt for this site returned 401/403. The site may be blocking automated access.",
		)
	case errors.Is(err, domain.ErrTimeout):
		return mcp.NewToolResultError(
			"The request timed out. The server may be slow or unreachable. Try again later.",
		)
	case errors.Is(err, domain.ErrContentTooLarge):
		return mcp.NewToolResultError(
			"The response exceeds the maximum size limit. Try fetching a smaller page or increasing the limit.",
		)
	case errors.Is(err, domain.ErrTooManyRedirects):
		return mcp.NewToolResultError(
			"Too many redirects. The URL may be part of a redirect loop.",
		)
	case errors.Is(err, domain.ErrInvalidURL) || errors.Is(err, domain.ErrSchemeNotAllowed):
		return mcp.NewToolResultError(
			fmt.Sprintf("Invalid URL: %s", err.Error()),
		)
	case errors.Is(err, domain.ErrFetchFailed):
		return mcp.NewToolResultError(
			fmt.Sprintf("Failed to fetch the URL: %s", err.Error()),
		)
	case errors.Is(err, domain.ErrExtractionFailed) || errors.Is(err, domain.ErrNoContent):
		return mcp.NewToolResultError(
			fmt.Sprintf("Content extraction failed: %s", err.Error()),
		)
	default:
		return mcp.NewToolResultError(err.Error())
	}
}

// handleFetchPrompt handles the fetch prompt.
func (s *Server) handleFetchPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	urlArg, ok := request.Params.Arguments["url"]
	if !ok || urlArg == "" {
		return nil, fmt.Errorf("URL argument is required")
	}

	req := domain.FetchRequest{
		URL: urlArg,
		Text: domain.TextOptions{
			MaxLength: 50000, // Larger limit for prompt-initiated fetches
		},
	}

	result, err := s.useCase.Fetch(ctx, req)
	if err != nil {
		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Failed to fetch %s", urlArg),
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: err.Error(),
					},
				},
			},
		}, nil
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Contents of %s", urlArg),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: s.formatResult(result),
				},
			},
		},
	}, nil
}

// formatResult formats the fetch result as text for the MCP response.
func (s *Server) formatResult(result *domain.FetchResult) string {
	text := ""
	if result.Title != "" {
		text += fmt.Sprintf("# %s\n\n", result.Title)
	}
	text += result.Content

	if result.RemainingContent > 0 {
		text += fmt.Sprintf("\n\n[Content truncated. %d characters remaining. Use start_index to continue.]", result.RemainingContent)
	}

	if result.RemainingImages > 0 {
		text += fmt.Sprintf("\n\n[%d more images available. Use imageStartIndex to continue.]", result.RemainingImages)
	}

	return text
}

// parseFetchRequest parses the MCP tool arguments into a domain.FetchRequest.
func parseFetchRequest(args map[string]interface{}) (*domain.FetchRequest, error) {
	req := &domain.FetchRequest{
		Text: domain.TextOptions{
			MaxLength: 20000,
			StartIndex: 0,
			Raw: false,
		},
		Images: domain.ImageOptions{
			Enable:           false,
			MaxCount:         3,
			StartIndex:       0,
			MaxWidth:         1000,
			MaxHeight:        1600,
			Quality:          80,
			AllowCrossOrigin: true,
			OutputBase64:     true,
			Layout:           "merged",
		},
		Security: domain.SecurityOptions{
			IgnoreRobotsTxt: true, // disabled by default for better UX
		},
	}

	// URL (required)
	urlVal, ok := args["url"].(string)
	if !ok || urlVal == "" {
		return nil, fmt.Errorf("url is required")
	}
	req.URL = urlVal

	// Text options
	if textObj, ok := args["text"].(map[string]interface{}); ok {
		if v, ok := textObj["maxLength"].(float64); ok {
			req.Text.MaxLength = int(v)
		}
		if v, ok := textObj["startIndex"].(float64); ok {
			req.Text.StartIndex = int(v)
		}
		if v, ok := textObj["raw"].(bool); ok {
			req.Text.Raw = v
		}
	}

	// Image options
	if imagesVal, ok := args["images"]; ok {
		switch v := imagesVal.(type) {
		case bool:
			req.Images.Enable = v
		case map[string]interface{}:
			req.Images.Enable = true
			if output, ok := v["output"].(string); ok {
				req.Images.OutputBase64 = output == "base64" || output == "both"
				req.Images.SaveToFile = output == "file" || output == "both"
			}
			if layout, ok := v["layout"].(string); ok {
				req.Images.Layout = layout
			}
			if maxCount, ok := v["maxCount"].(float64); ok {
				req.Images.MaxCount = int(maxCount)
			}
			if startIndex, ok := v["startIndex"].(float64); ok {
				req.Images.StartIndex = int(startIndex)
			}
			if size, ok := v["size"].(map[string]interface{}); ok {
				if w, ok := size["maxWidth"].(float64); ok {
					req.Images.MaxWidth = int(w)
				}
				if h, ok := size["maxHeight"].(float64); ok {
					req.Images.MaxHeight = int(h)
				}
				if q, ok := size["quality"].(float64); ok {
					req.Images.Quality = int(q)
				}
			}
			if policy, ok := v["originPolicy"].(string); ok {
				req.Images.AllowCrossOrigin = policy == "cross-origin"
			}
		}
	}

	// Security options
	if secObj, ok := args["security"].(map[string]interface{}); ok {
		if v, ok := secObj["ignoreRobotsTxt"].(bool); ok {
			req.Security.IgnoreRobotsTxt = v
		}
	}

	return req, nil
}

// ServeStdio starts the MCP server on stdio transport.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// MCPServer returns the underlying mcp-go server for testing.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}
