package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGentypes(t *testing.T) {
	tempDir := t.TempDir()
	outFile := filepath.Join(tempDir, "types.ts")

	// Set os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{
		"gentypes",
		"-models", "../../internal/model",
		"-out", outFile,
	}

	main()

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read generated output: %v", err)
	}

	content := string(data)
	for _, expectedType := range []string{"BoardState", "Task", "Milestone", "Strategy", "GlossaryTerm", "Priority", "SSEEvent"} {
		if !strings.Contains(content, expectedType) {
			t.Errorf("expected generated output to contain %q", expectedType)
		}
	}
}
