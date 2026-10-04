package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCleanURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"https://github.com/preactjs/signals", "https://github.com/preactjs/signals"},
		{"http://github.com/preactjs/signals", "https://github.com/preactjs/signals"},
		{"git+https://github.com/braintree/sanitize-url.git", "https://github.com/braintree/sanitize-url"},
		{"git://github.com/cure53/DOMPurify.git", "https://github.com/cure53/DOMPurify"},
		{"github:fabiospampinato/khroma", "https://github.com/fabiospampinato/khroma"},
		{"github.com/markedjs/marked", "https://github.com/markedjs/marked"},
		{"preactjs/preact", "https://github.com/preactjs/preact"},
		{"lodash/lodash", "https://github.com/lodash/lodash"},
		{"ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"git@github.com:owner/repo.git", "https://github.com/owner/repo"},
		{"https://valibot.dev", "https://valibot.dev"},
	}

	for _, tt := range tests {
		actual := cleanURL(tt.input)
		if actual != tt.expected {
			t.Errorf("cleanURL(%q) = %q, want %q", tt.input, actual, tt.expected)
		}
	}
}

func TestIsExcludedNpmPackage(t *testing.T) {
	tests := []struct {
		pkg      string
		expected bool
	}{
		{"@types/d3", true},
		{"@types/geojson", true},
		{"tailwindcss", true},
		{"daisyui", true},
		{"bun-plugin-tailwind", true},
		{"@tailwindcss/typography", true},
		{"preact", false},
		{"@preact/signals", false},
		{"valibot", false},
		{"mermaid", false},
	}

	for _, tt := range tests {
		actual := isExcludedNpmPackage(tt.pkg)
		if actual != tt.expected {
			t.Errorf("isExcludedNpmPackage(%q) = %v, want %v", tt.pkg, actual, tt.expected)
		}
	}
}

func TestExtractNpmURL(t *testing.T) {
	// String repository
	u1 := extractNpmURL("preactjs/preact", "")
	if u1 != "https://github.com/preactjs/preact" {
		t.Errorf("expected https://github.com/preactjs/preact, got %q", u1)
	}

	// Map repository
	repoMap := map[string]any{"type": "git", "url": "git+https://github.com/braintree/sanitize-url.git"}
	u2 := extractNpmURL(repoMap, "")
	if u2 != "https://github.com/braintree/sanitize-url" {
		t.Errorf("expected https://github.com/braintree/sanitize-url, got %q", u2)
	}

	// Fallback to homepage
	u3 := extractNpmURL(nil, "https://valibot.dev")
	if u3 != "https://valibot.dev" {
		t.Errorf("expected https://valibot.dev, got %q", u3)
	}
}

func TestMainNpmPackages(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
	"dependencies": { "preact": "11.0.0", "mermaid": "^12", "lucide-preact": "^1" },
	"devDependencies": { "lighthouse": "^13" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := mainNpmPackages(dir)
	if err != nil {
		t.Fatalf("mainNpmPackages: %v", err)
	}
	want := []string{"lucide-preact", "mermaid", "preact"}
	if !slices.Equal(got, want) {
		t.Errorf("mainNpmPackages = %v, want %v (dependencies only, sorted)", got, want)
	}
}
