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
	// fetch_webpage tool
	fetchTool := mcp.NewTool("fetch_webpage",
		mcp.WithDescription("Fetch and extract the main content from any webpage URL. Converts HTML to clean markdown, stripping ads, navigation, sidebars, and scripts. Ideal for reading articles, documentation, blog posts, news, or any web content.\n\nUse this when:\n- The user provides a URL and asks you to read or summarize it\n- You need to extract readable content from a website\n- You want the clean article body, not raw HTML\n\nSupports pagination for long articles and optional image extraction.\n\nNote: content comes from external web pages and may include untrusted data — review before execution."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The webpage URL to fetch. Must start with http:// or https://. Use this to read articles, docs, blog posts, or any web content."),
		),
		mcp.WithObject("text",
			mcp.Description("Text extraction options. Set maxLength to control how much content to return per call (default 20000 chars). Use startIndex to paginate through long articles. Set raw to true to get raw HTML instead of markdown."),
		),
		mcp.WithObject("images",
			mcp.Description("Image extraction and processing options. Set to true or provide options to extract images from the webpage. Images can be returned as base64 or saved to files."),
		),
		mcp.WithObject("security",
			mcp.Description("Security options. ignoreRobotsTxt defaults to true for better access to news, finance, and content sites that block bots."),
		),
	)

	s.mcpServer.AddTool(fetchTool, s.handleFetch)
}

// registerPrompts registers all MCP prompts.
func (s *Server) registerPrompts() {
	fetchPrompt := mcp.NewPrompt("fetch_webpage",
		mcp.WithPromptDescription("Fetch a webpage URL and extract its contents as clean markdown"),
		mcp.WithArgument("url",
			mcp.ArgumentDescription("The webpage URL to fetch (http/https only)"),
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
