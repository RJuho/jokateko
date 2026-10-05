package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/version"
)

func TestCLI_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "jokateko") {
		t.Errorf("expected version output to mention jokateko, got: %q", stdout.String())
	}

	// JSON format
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"version", "-json"}, &stdout, &stderr)
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
	code := run(t.Context(), []string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Available Commands:") {
		t.Errorf("expected help output, got: %q", stdout.String())
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"nonexistent-cmd"}, &stdout, &stderr)
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
	code := run(t.Context(), []string{"init", "-dir", tempDir, "-name", "CLI Project"}, &stdout, &stderr)
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

	// Verify that the generated config.toml has only [project] active and other sections commented
	cfgBytes, err := os.ReadFile(filepath.Join(tempDir, ".jokateko", "config.toml"))
	if err != nil {
		t.Fatalf("failed to read generated config.toml: %v", err)
	}
	cfgStr := string(cfgBytes)
	if !strings.Contains(cfgStr, `name = "CLI Project"`) {
		t.Errorf("expected generated config.toml to contain project name, got:\n%s", cfgStr)
	}
	if !strings.Contains(cfgStr, "# [server]") || !strings.Contains(cfgStr, "# [[board.columns]]") {
		t.Errorf("expected [server] and [[board.columns]] sections to be commented out, got:\n%s", cfgStr)
	}

	// 2. Run 'parse' on initialized project
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"parse", "-dir", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("parse failed with code %d. stderr:\n%s\nstdout:\n%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "[OK] Validation successful") {
		t.Errorf("expected validation success message, got:\n%s", stdout.String())
	}

	// 3. Run 'lint' (alias for parse)
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"lint", "-dir", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint alias failed with code %d", code)
	}
}

func TestCLI_Build(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize workspace first
	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), []string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	outFile := filepath.Join(tempDir, "dist", "index.html")
	code := run(t.Context(), []string{"build", "-dir", tempDir, "-out", outFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("build failed with exit code %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Exported self-contained Kanban snapshot") {
		t.Errorf("expected success message in stdout, got:\n%s", stdout.String())
	}

	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatalf("expected output file to exist at %s", outFile)
	}
	if !strings.Contains(stdout.String(), "mermaidjs=cdn") {
		t.Errorf("expected default mermaidjs=cdn in stdout, got:\n%s", stdout.String())
	}

	// --mermaidjs accepts bundled and rejects unknown modes
	stdout.Reset()
	stderr.Reset()
	if code := run(t.Context(), []string{"build", "-dir", tempDir, "-out", outFile, "--mermaidjs=bundled"}, &stdout, &stderr); code != 0 {
		t.Fatalf("build --mermaidjs=bundled failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "mermaidjs=bundled") {
		t.Errorf("expected mermaidjs=bundled in stdout, got:\n%s", stdout.String())
	}

	stderr.Reset()
	if code := run(t.Context(), []string{"build", "-dir", tempDir, "-out", outFile, "--mermaidjs=inline"}, &stdout, &stderr); code == 0 {
		t.Fatal("expected build to fail for invalid --mermaidjs value")
	}
	if !strings.Contains(stderr.String(), "invalid mermaidjs mode") {
		t.Errorf("expected invalid mode error, got: %s", stderr.String())
	}
}

func TestCLI_Serve(t *testing.T) {
	tempDir := t.TempDir()

	// Initialize workspace first
	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), []string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Run serve with port 0 (dynamic port) and 150ms timeout
	code := run(t.Context(), []string{"serve", "-dir", tempDir, "-port", "0", "-timeout", "150ms"}, &stdout, &stderr)
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
	if code := run(t.Context(), []string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	// Run mcp with 150ms timeout (standalone fallback)
	code := runMCP(t.Context(), []string{"-dir", tempDir, "-timeout", "150ms"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runMCP failed with exit code %d. stderr: %s", code, stderr.String())
	}

	if !strings.Contains(stderr.String(), "standalone") {
		t.Errorf("expected standalone mode notice in stderr, got:\n%s", stderr.String())
	}
}

func TestCLI_InitReplace(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initial init
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"init", "-dir", tempDir, "-name", "Original Name"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("initial init failed: %s", stderr.String())
	}

	configFile := filepath.Join(tempDir, ".jokateko", "config.toml")
	// Modify the config file with custom text
	customText := `# Custom User Config
version = "0"
[project]
name = "Modified Custom Name"
`
	if err := os.WriteFile(configFile, []byte(customText), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Running init without --replace should NOT overwrite the config
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"init", "-dir", tempDir, "-name", "Ignored Name"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second init failed: %s", stderr.String())
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != customText {
		t.Errorf("expected config not to be overwritten without --replace, got:\n%s", string(content))
	}

	// 3. Running init with --replace should overwrite the config with fresh default template
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"init", "-dir", tempDir, "-name", "Replaced Name", "--replace"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init with --replace failed: %s", stderr.String())
	}

	replacedContent, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	replacedStr := string(replacedContent)
	if !strings.Contains(replacedStr, `name = "Replaced Name"`) {
		t.Errorf("expected config to be overwritten with 'Replaced Name', got:\n%s", replacedStr)
	}
	if !strings.Contains(replacedStr, "# [server]") || !strings.Contains(replacedStr, "# [[board.columns]]") {
		t.Errorf("expected fresh commented config template upon --replace, got:\n%s", replacedStr)
	}
}

func TestCLI_About(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"about"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Jokateko") {
		t.Errorf("expected about output to contain Jokateko, got: %s", out)
	}
	if !strings.Contains(out, "MIT License") {
		t.Errorf("expected about output to contain MIT License, got: %s", out)
	}
	if !strings.Contains(out, "jokateko licenses") {
		t.Errorf("expected about output to cross-reference 'jokateko licenses', got: %s", out)
	}

	// JSON format
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"about", "-json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for about -json, got %d. stderr: %s", code, stderr.String())
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse about json: %v", err)
	}
	if parsed["name"] != "Jokateko" || parsed["license"] != "MIT" {
		t.Errorf("unexpected about JSON output: %+v", parsed)
	}
}

func TestCLI_Licenses(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(t.Context(), []string{"licenses"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Jokateko Open Source Licenses") {
		t.Errorf("expected licenses header, got: %s", out)
	}
	if !strings.Contains(out, "Go Backend Dependencies") || !strings.Contains(out, "Web Frontend Dependencies") {
		t.Errorf("expected Go and Web dependencies sections, got: %s", out)
	}
	if !strings.Contains(out, "jokateko about") {
		t.Errorf("expected cross-reference to 'jokateko about', got: %s", out)
	}

	// Full flag format
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"licenses", "-full"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for licenses -full, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "JOKATEKO (MIT)") {
		t.Errorf("expected full license header, got: %s", stdout.String())
	}

	// JSON format
	stdout.Reset()
	stderr.Reset()
	code = run(t.Context(), []string{"licenses", "-json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for licenses -json, got %d. stderr: %s", code, stderr.String())
	}

	var parsed struct {
		Project  version.ProjectLicense   `json:"project"`
		Packages []version.PackageLicense `json:"packages"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse licenses json: %v", err)
	}
	if parsed.Project.License != "MIT" {
		t.Errorf("expected project license to be MIT, got %s", parsed.Project.License)
	}
	if len(parsed.Packages) == 0 {
		t.Errorf("expected packages array not to be empty")
	}
}

func TestCLI_MCPDisabled(t *testing.T) {
	tempDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), []string{"init", "-dir", tempDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("failed to init workspace: %s", stderr.String())
	}
	cfgPath := filepath.Join(tempDir, ".jokateko", "config.toml")
	f, err := os.OpenFile(cfgPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n[mcp]\nenabled = false\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	stderr.Reset()
	if code := runMCP(t.Context(), []string{"-dir", tempDir, "-timeout", "150ms"}, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("runMCP exit code = %d, want 1. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "MCP is disabled") {
		t.Errorf("expected disabled message, got: %s", stderr.String())
	}
}
