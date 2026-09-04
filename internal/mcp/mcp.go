// Package mcp provides the Model Context Protocol (MCP) server implementation for Jokateko,
// exposing structured tools, resources, and prompts for AI coding agents.
package mcp

import (
	"path/filepath"
	"strings"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/version"
	"github.com/RJuho/jokateko/internal/writer"
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
	"5. All modifications to .jokateko/ files must be performed via Jokateko MCP tools rather than direct file edits, ensuring atomic disk persistence and schema validation."

// Server encapsulates the Jokateko Model Context Protocol server.
type Server struct {
	mcpServer    *mcp.Server
	cfg          *config.Config
	workspaceDir string
	store        *store.Store
	writer       *writer.Writer
	instructions string
}

// New initializes an MCP server with all Jokateko tools, resources, and prompt templates.
func New(cfg *config.Config, workspaceDir string, st *store.Store, wr *writer.Writer) *Server {
	if cfg == nil {
		cfg = config.Default(workspaceDir)
	}

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

	opts := &mcp.ServerOptions{
		Instructions: instructions,
	}

	server := &Server{
		mcpServer:    mcp.NewServer(impl, opts),
		cfg:          cfg,
		workspaceDir: workspaceDir,
		store:        st,
		writer:       wr,
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

// TasksDir returns the absolute path to the tasks directory.
func (s *Server) TasksDir() string {
	p := s.cfg.Paths.Tasks
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.workspaceDir, p)
}

// MilestonesDir returns the absolute path to the milestones directory.
func (s *Server) MilestonesDir() string {
	p := s.cfg.Paths.Milestones
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.workspaceDir, p)
}

// StrategiesDir returns the absolute path to the strategies directory.
func (s *Server) StrategiesDir() string {
	p := s.cfg.Paths.Strategies
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.workspaceDir, p)
}

// GlossaryDir returns the absolute path to the glossary directory.
func (s *Server) GlossaryDir() string {
	p := s.cfg.Paths.Glossary
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(s.workspaceDir, p)
}

// Instructions returns the active MCP server system instructions string.
func (s *Server) Instructions() string {
	return s.instructions
}

