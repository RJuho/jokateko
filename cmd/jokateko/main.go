// Package main is the entrypoint for the Jokateko CLI binary.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/RJuho/jokateko/internal/version"
)

func main() {
	// A single signal context drives graceful shutdown of long-running commands,
	// so their deferred cleanup always runs.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return cmdServe(ctx, nil, stdout, stderr)
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "serve":
		return cmdServe(ctx, subArgs, stdout, stderr)
	case "parse", "lint":
		return cmdParse(subArgs, stdout, stderr)
	case "init":
		return cmdInit(subArgs, stdout, stderr)
	case "build":
		return cmdBuild(subArgs, stdout, stderr)
	case "mcp":
		return runMCP(ctx, subArgs, os.Stdin, stdout, stderr)
	case "version", "-v", "--version", "-version":
		return cmdVersion(subArgs, stdout, stderr)
	case "about":
		return cmdAbout(subArgs, stdout, stderr)
	case "licenses", "license":
		return cmdLicenses(subArgs, stdout, stderr)
	case "help", "-h", "--help", "-help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %q\nRun 'jokateko help' for usage.\n", cmd)
		return 1
	}
}

// requireDir fails when a -dir workspace argument does not name an existing
// directory, so a mistyped path is reported instead of silently treated as an
// empty project.
func requireDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("workspace directory %q does not exist", dir)
		}
		return fmt.Errorf("cannot access workspace directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace path %q is not a directory", dir)
	}
	return nil
}

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `Jokateko - Markdown-driven Kanban & task management for developers and AI agents

Usage:
  jokateko [command]

Available Commands:
  serve       Start daemon, file watcher, local Web UI, and MCP server (default)
  parse       Dry-run validate configuration, markdown files, and task dependency DAG (alias: lint)
  init        Scaffold a new .jokateko/ directory and default configuration
  build       Export a single-file static HTML snapshot (--mermaidjs=cdn|bundled|none)
  mcp         Run MCP stdio server or proxy to active serve daemon
  version     Print binary version, commit, date, and architecture
  about       Display project information, build metadata, and MIT license
  licenses    List third-party open-source dependencies and licenses
  help        Display this help message

Flags:
  Run 'jokateko [command] --help' for command-specific flags.

Version: %s
`, version.Get())
}
