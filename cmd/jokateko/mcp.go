package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/proxy"
)

func cmdMCP(args []string, stdout, stderr io.Writer) int {
	return runMCP(args, os.Stdin, stdout, stderr)
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", ".", "Project root directory")
	portFlag := fs.Int("port", -1, "Daemon port to probe (overrides config.toml)")
	timeoutFlag := fs.Duration("timeout", 0, "Automatically shut down after duration (useful for smoke testing)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	workspaceDir, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid workspace directory: %v\n", err)
		return 1
	}

	cfg, err := config.Load(workspaceDir)
	if err != nil {
		cfg = config.Default(workspaceDir)
	}

	if *portFlag > 0 {
		cfg.Server.Port = *portFlag
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *timeoutFlag > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, *timeoutFlag)
		defer timeoutCancel()
	}

	// Listen for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	runner := proxy.NewRunner(cfg, workspaceDir, stdin, stdout, stderr)
	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, io.EOF) {
		fmt.Fprintf(stderr, "error running MCP: %v\n", err)
		return 1
	}

	return 0
}
