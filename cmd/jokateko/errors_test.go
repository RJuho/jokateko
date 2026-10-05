package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initWorkspace scaffolds a fresh project in a temp dir and returns its path.
func initWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), []string{"init", "-dir", dir}, &stdout, &stderr); code != 0 {
		t.Fatalf("init failed with code %d: %s", code, stderr.String())
	}
	return dir
}

// writeConfig overwrites the workspace's .jokateko/config.toml.
func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".jokateko", "config.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCLI_ParseErrors(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) []string
		wantCode   int
		wantStderr string
		wantStdout string
	}{
		{
			name:       "unknown flag",
			setup:      func(*testing.T) []string { return []string{"-bogus"} },
			wantCode:   1,
			wantStderr: "flag provided but not defined",
		},
		{
			name: "invalid TOML config",
			setup: func(t *testing.T) []string {
				dir := initWorkspace(t)
				writeConfig(t, dir, "[project\nname = ")
				return []string{"-dir", dir}
			},
			wantCode:   1,
			wantStderr: "CFG-001",
		},
		{
			name: "config failing schema validation",
			setup: func(t *testing.T) []string {
				dir := initWorkspace(t)
				writeConfig(t, dir, "[paths]\ntasks = \"../outside\"\n")
				return []string{"-dir", dir}
			},
			wantCode:   1,
			wantStderr: "CFG-005",
		},
		{
			name: "missing workspace directory",
			setup: func(t *testing.T) []string {
				return []string{"-dir", filepath.Join(t.TempDir(), "does-not-exist")}
			},
			wantCode:   0,
			wantStdout: "[OK] Validation successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.setup(t)
			var stdout, stderr bytes.Buffer
			code := run(t.Context(), append([]string{"parse"}, args...), &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tt.wantCode, stdout.String(), stderr.String())
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr missing %q, got:\n%s", tt.wantStderr, stderr.String())
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout missing %q, got:\n%s", tt.wantStdout, stdout.String())
			}
		})
	}
}

func TestCLI_BuildPathsAndErrors(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (args []string, wantFile string)
		wantCode   int
		wantStderr string
	}{
		{
			name: "unknown flag",
			setup: func(*testing.T) ([]string, string) {
				return []string{"-bogus"}, ""
			},
			wantCode:   1,
			wantStderr: "flag provided but not defined",
		},
		{
			name: "invalid TOML config falls back to default export path",
			setup: func(t *testing.T) ([]string, string) {
				dir := initWorkspace(t)
				writeConfig(t, dir, "[project\nname = ")
				return []string{"-dir", dir}, filepath.Join(dir, "dist-kanban", "index.html")
			},
			wantCode: 0,
		},
		{
			name: "configured export directory without extension gets index.html",
			setup: func(t *testing.T) ([]string, string) {
				dir := initWorkspace(t)
				writeConfig(t, dir, "[paths]\nexport = \"site\"\n")
				return []string{"-dir", dir}, filepath.Join(dir, "site", "index.html")
			},
			wantCode: 0,
		},
		{
			name: "relative -out is resolved against the workspace",
			setup: func(t *testing.T) ([]string, string) {
				dir := initWorkspace(t)
				return []string{"-dir", dir, "-out", "snap/out.html"}, filepath.Join(dir, "snap", "out.html")
			},
			wantCode: 0,
		},
		{
			name: "missing workspace directory still exports an empty board",
			setup: func(t *testing.T) ([]string, string) {
				dir := filepath.Join(t.TempDir(), "does-not-exist")
				return []string{"-dir", dir}, filepath.Join(dir, "dist-kanban", "index.html")
			},
			wantCode: 0,
		},
		{
			name: "output path below a regular file fails to export",
			setup: func(t *testing.T) ([]string, string) {
				dir := initWorkspace(t)
				blocker := filepath.Join(dir, "blocker")
				if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
				return []string{"-dir", dir, "-out", filepath.Join(blocker, "index.html")}, ""
			},
			wantCode:   1,
			wantStderr: "failed to export snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, wantFile := tt.setup(t)
			var stdout, stderr bytes.Buffer
			code := run(t.Context(), append([]string{"build", "--mermaidjs=none"}, args...), &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tt.wantCode, stdout.String(), stderr.String())
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr missing %q, got:\n%s", tt.wantStderr, stderr.String())
			}
			if wantFile != "" {
				if _, err := os.Stat(wantFile); err != nil {
					t.Errorf("expected export at %s: %v", wantFile, err)
				}
				if !strings.Contains(stdout.String(), wantFile) {
					t.Errorf("stdout should name %s, got:\n%s", wantFile, stdout.String())
				}
			}
		})
	}
}
