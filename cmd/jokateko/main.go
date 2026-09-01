// Package main is the entrypoint for the Jokateko CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/RJuho/jokateko/internal/version"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "version", "-v", "--version", "-version":
			fmt.Println(version.Get())
			return
		case "help", "-h", "--help", "-help":
			printUsage()
			return
		}
	}

	// Default behavior until subsequent phases implement subcommands
	if len(args) == 0 {
		printUsage()
		return
	}

	fmt.Fprintf(os.Stderr, "unknown command: %q\nRun 'jokateko help' for usage.\n", args[0])
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`Jokateko - Markdown-driven Kanban & task management for developers and AI agents

Usage:
  jokateko [command]

Available Commands:
  serve       Start daemon, file watcher, local Web UI, and MCP server
  parse       Dry-run validate configuration, markdown files, and task dependency DAG
  build       Export self-contained static HTML offline snapshot
  init        Scaffold a new .jokateko/ directory and default configuration
  mcp         Run MCP stdio server or proxy to active serve daemon
  version     Print binary version, commit, date, and architecture
  help        Display this help message

Version: %s
`, version.Get())
}
