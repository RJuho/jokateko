// Package main is the entrypoint for the Jokateko CLI binary.
package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/RJuho/jokateko/internal/version"
)

func main() {
	// Set up top-level signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Exit(0)
	}()

	code := run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return cmdServe(nil, stdout, stderr)
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "serve":
		return cmdServe(subArgs, stdout, stderr)
	case "parse", "lint":
		return cmdParse(subArgs, stdout, stderr)
	case "init":
		return cmdInit(subArgs, stdout, stderr)
	case "build":
		return cmdBuild(subArgs, stdout, stderr)
	case "mcp":
		return cmdMCP(subArgs, stdout, stderr)
	case "version", "-v", "--version", "-version":
		return cmdVersion(subArgs, stdout, stderr)
	case "help", "-h", "--help", "-help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %q\nRun 'jokateko help' for usage.\n", cmd)
		return 1
	}
}

func cmdBuild(_ []string, stdout, _ io.Writer) int {
	// Placeholder until Phase 8 (Self-Contained Static HTML Exporter)
	fmt.Fprintln(stdout, "Jokateko build: exporting static HTML offline snapshot (activated in Phase 8)")
	return 0
}

func cmdMCP(_ []string, stdout, _ io.Writer) int {
	// Placeholder until Phase 9 (Model Context Protocol Integration)
	fmt.Fprintln(stdout, "Jokateko mcp: stdio protocol server (activated in Phase 9)")
	return 0
}

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Jokateko - Markdown-driven Kanban & task management for developers and AI agents

Usage:
  jokateko [command]

Available Commands:
  serve       Start daemon, file watcher, local Web UI, and MCP server (default)
  parse       Dry-run validate configuration, markdown files, and task dependency DAG (alias: lint)
  init        Scaffold a new .jokateko/ directory and default configuration
  build       Export self-contained static HTML offline snapshot
  mcp         Run MCP stdio server or proxy to active serve daemon
  version     Print binary version, commit, date, and architecture
  help        Display this help message

Flags:
  Run 'jokateko [command] --help' for command-specific flags.

Version: %s
`, version.Get())
}
