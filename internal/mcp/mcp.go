// Package mcp provides the Model Context Protocol (MCP) server implementation for Jokateko,
// exposing structured tools, resources, and prompts for AI coding agents.
package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultInstructions is the standard system guidance provided to AI agents
// when no custom instructions are configured in .jokateko/config.toml.
const DefaultInstructions = "Jokateko manages tasks, milestones, architectural strategies, and glossary terms " +
	"as version-controlled Markdown files in the repository (.jokateko/).\n\n" +
	"Core Workflow Rules:\n" +
	"1. Always use get_task to inspect full acceptance criteria before starting work.\n" +
	"2. Use update_task_item to toggle checklists as items are completed.\n" +
	"3. Use complete_task when finished to record completion summary and unblock downstream tasks.\n" +
	"4. Always consult Tier-1 architectural strategies via get_strategy before making architectural decisions.\n" +
	"5. All modifications to .jokateko/ files must be performed via Jokateko MCP tools rather than direct file edits, ensuring atomic disk persistence and schema validation.\n" +
	"6. Never edit files in .jokateko/ directly with file writing tools; direct writes bypass schema validation, cycle checks, and live UI event broadcasting."

// DeleteEntityOutput represents the standard result of an entity deletion operation.
type DeleteEntityOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// Server encapsulates the Jokateko Model Context Protocol server.
type Server struct {
	mcpServer    *mcp.Server
	cfg          *config.Config
	svc          *service.Service
	store        *store.Store
	instructions string
}

func formatWorkflowGuidance(cols []config.ColumnConfig) string {
	var lines []string
	for _, c := range cols {
		handledBy := strings.TrimSpace(c.HandledBy)
		instr := strings.TrimSpace(c.Instructions)
		if handledBy != "" || instr != "" {
			line := fmt.Sprintf("- %s (`%s`):", c.Name, c.ID)
			if handledBy != "" {
				line += fmt.Sprintf(" Handled by %s.", handledBy)
			}
			if instr != "" {
				line += fmt.Sprintf(" %s", instr)
			}
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\nWorkflow Column Ownership & Instructions:\n" + strings.Join(lines, "\n")
}

// New initializes an MCP server with all Jokateko tools, resources, and prompt templates.
func New(svc *service.Service) *Server {
	cfg := svc.Config()

	impl := &mcp.Implementation{
		Name:        "jokateko",
		Title:       "Jokateko Kanban & Spec-First Engine",
		Description: "Local, Markdown-driven Kanban and architectural guidance engine for AI agents",
		Version:     version.Get().Version,
	}

	instructions := strings.TrimSpace(cfg.MCP.Instructions)
	if instructions == "" {
		instructions = DefaultInstructions
	}

	if guidance := formatWorkflowGuidance(cfg.Board.Columns); guidance != "" {
		instructions += guidance
	}

	opts := &mcp.ServerOptions{
		Instructions: instructions,
	}

	server := &Server{
		mcpServer:    mcp.NewServer(impl, opts),
		cfg:          cfg,
		svc:          svc,
		store:        svc.Store(),
		instructions: instructions,
	}

	// Register tools
	server.registerTaskTools()
	server.registerMilestoneTools()
	server.registerStrategyTools()
	server.registerGlossaryTools()
	server.registerSearchTools()
	server.registerTagTools()
	server.registerBoardTools()

	// Register resources & prompts
	server.registerResources()
	server.registerPrompts()

	return server
}

// MCPServer returns the underlying official SDK MCP server instance.
func (s *Server) MCPServer() *mcp.Server {
	return s.mcpServer
}

// Instructions returns the active MCP server system instructions string.
func (s *Server) Instructions() string {
	return s.instructions
}

var errMutationsDisabled = errors.New("mutations are disabled in configuration")

// checkMutations rejects write tools when [mcp] allow_mutations is false.
func (s *Server) checkMutations() error {
	if !s.cfg.MCP.AllowMutations {
		return errMutationsDisabled
	}
	return nil
}

// requireID trims id and returns an error naming kind when it is empty.
func requireID(id, kind string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%s id is required", kind)
	}
	return id, nil
}

// HTTPHandler returns the Streamable HTTP transport handler for this server.
//
// The server keeps no per-session state, so it runs in stateless mode: this is
// required for clients speaking the sessionless 2026-07-28 protocol revision,
// while older clients still complete the legacy initialize handshake. The SDK's
// default localhost (DNS rebinding) protection stays enabled.
func (s *Server) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s.mcpServer
	}, &mcp.StreamableHTTPOptions{Stateless: true})
}
