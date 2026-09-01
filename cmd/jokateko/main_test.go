package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "jokateko") {
		t.Errorf("expected version output to mention jokateko, got: %q", stdout.String())
	}

	// JSON format
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"version", "-json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse version json: %v, raw: %s", err, stdout.String())
	}
	if _, ok := parsed["version"]; !ok {
		t.Errorf("expected 'version' key in JSON output: %+v", parsed)
	}
}

func TestCLI_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Available Commands:") {
		t.Errorf("expected help output, got: %q", stdout.String())
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"nonexistent-cmd"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("expected error message in stderr, got: %q", stderr.String())
	}
}

func TestCLI_InitAndParseCycle(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Run 'init'
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", "-dir", tempDir, "-name", "CLI Project"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init failed with code %d. stderr: %s", code, stderr.String())
	}

	// Check files created
	expectedPaths := []string{
		filepath.Join(tempDir, ".jokateko", "config.toml"),
		filepath.Join(tempDir, ".jokateko", "tasks"),
		filepath.Join(tempDir, ".jokateko", "milestones"),
		filepath.Join(tempDir, ".jokateko", "strategies"),
		filepath.Join(tempDir, ".jokateko", "glossary"),
	}
	for _, p := range expectedPaths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected path %q to exist after init", p)
		}
	}

	// 2. Run 'parse' on initialized project
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"parse", "-dir", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("parse failed with code %d. stderr:\n%s\nstdout:\n%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "[OK] Validation successful") {
		t.Errorf("expected validation success message, got:\n%s", stdout.String())
	}

	// 3. Run 'lint' (alias for parse)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"lint", "-dir", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint alias failed with code %d", code)
	}
}

func TestCLI_Build(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize workspace first
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	outFile := filepath.Join(tempDir, "dist", "index.html")
	code := run([]string{"build", "-dir", tempDir, "-out", outFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build failed with exit code %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Exported self-contained Kanban snapshot") {
		t.Errorf("expected success message in stdout, got:\n%s", stdout.String())
	}

	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatalf("expected output file to exist at %s", outFile)
	}
}

func TestCLI_Serve(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize workspace first
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Run serve with port 0 (dynamic port) and 150ms timeout
	code := run([]string{"serve", "-dir", tempDir, "-port", "0", "-timeout", "150ms"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("serve failed with exit code %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Jokateko daemon active") {
		t.Errorf("expected daemon active output, got:\n%s", stdout.String())
	}
}

func TestCLI_MCP(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize workspace first
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Run mcp with 150ms timeout (standalone fallback)
	code := runMCP([]string{"-dir", tempDir, "-timeout", "150ms"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runMCP failed with exit code %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stderr.String(), "standalone") {
		t.Errorf("expected standalone mode notice in stderr, got:\n%s", stderr.String())
	}
}

