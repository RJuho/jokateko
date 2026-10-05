package main

// Not covered here: main, directGoModules and harvestGoLicenses. They shell out
// to `go mod edit -json` and `go list -deps ./cmd/jokateko` and depend on the
// repository's go.mod and module cache, so a unit test would be slow,
// environment-dependent and would not exercise anything the pure helpers below
// don't already check. `make generate` exercises them end to end.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSPDX(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"empty", "", "Unknown"},
		{"MIT header", "The MIT License (MIT)\nCopyright (c) 2024", "MIT"},
		{"MIT body only", "Permission is hereby granted, free of charge, to any person", "MIT"},
		{"Apache-2.0", "Apache License\nVersion 2.0, January 2004", "Apache-2.0"},
		{"Apache without version is not Apache-2.0", "Apache License", "Unknown"},
		{"BSD-3 header", "BSD 3-Clause License", "BSD-3-Clause"},
		{"BSD-3 body", "Redistribution and use in source and binary forms ... Neither the name of Google", "BSD-3-Clause"},
		{"BSD-2", "BSD 2-Clause License", "BSD-2-Clause"},
		{"MPL-2.0", "Mozilla Public License Version 2.0", "MPL-2.0"},
		{"ISC header", "ISC License\nCopyright (c) Isaac", "ISC"},
		{"ISC body", "Permission to use, copy, modify, and/or distribute this software for any purpose", "ISC"},
		{"Unlicense", "This is free and unencumbered software released into the public domain.", "Unlicense"},
		{"CC0", "Creative Commons Zero v1.0 Universal", "CC0-1.0"},
		{"case insensitive", "THE MIT LICENSE", "MIT"},
		{"unknown text", "All rights reserved. Proprietary.", "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectSPDX(tt.content); got != tt.want {
				t.Errorf("detectSPDX(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestExtractNpmLicense(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", "MIT", "MIT"},
		{"object with type", map[string]any{"type": "ISC", "url": "https://example.com"}, "ISC"},
		{"object without type", map[string]any{"url": "https://example.com"}, ""},
		{"object with non-string type", map[string]any{"type": 42.0}, ""},
		{"nil", nil, ""},
		{"array (legacy licenses field)", []any{"MIT", "Apache-2.0"}, ""},
		{"number", 1.0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractNpmLicense(tt.in); got != tt.want {
				t.Errorf("extractNpmLicense(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGoModuleURL(t *testing.T) {
	tests := []struct {
		mod  string
		want string
	}{
		{"github.com/pelletier/go-toml/v2", "https://github.com/pelletier/go-toml/v2"},
		{"golang.org/x/sync", "https://cs.opensource.google/go/x/sync"},
		{"modernc.org/sqlite", "https://modernc.org/sqlite"},
		{"gopkg.in/yaml.v3", "https://gopkg.in/yaml.v3"},
	}
	for _, tt := range tests {
		t.Run(tt.mod, func(t *testing.T) {
			if got := goModuleURL(tt.mod); got != tt.want {
				t.Errorf("goModuleURL(%q) = %q, want %q", tt.mod, got, tt.want)
			}
		})
	}
}

// writeFiles creates name→content files (and "name/" directories) under dir.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if name[len(name)-1] == '/' {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFindLicenseInDir(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantFile string
		wantText string
	}{
		{"LICENSE", map[string]string{"LICENSE": "mit", "README.md": "readme"}, "LICENSE", "mit"},
		{"LICENSE.md", map[string]string{"LICENSE.md": "md"}, "LICENSE.md", "md"},
		{"COPYING", map[string]string{"COPYING": "gpl"}, "COPYING", "gpl"},
		{"British spelling", map[string]string{"Licence.txt": "uk"}, "Licence.txt", "uk"},
		{"preferred order wins", map[string]string{"COPYING": "second", "license": "first"}, "license", "first"},
		{"prefix fallback", map[string]string{"LICENSE-MIT": "fallback"}, "LICENSE-MIT", "fallback"},
		{"directory named LICENSE is skipped", map[string]string{"LICENSE/": "", "LICENSE-APACHE": "apache"}, "LICENSE-APACHE", "apache"},
		{"directory with license prefix is skipped", map[string]string{"licenses/": ""}, "", ""},
		{"none", map[string]string{"README.md": "readme", "main.go": "package main"}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tt.files)
			gotFile, gotText := findLicenseInDir(dir)
			if gotFile != tt.wantFile || gotText != tt.wantText {
				t.Errorf("findLicenseInDir = (%q, %q), want (%q, %q)", gotFile, gotText, tt.wantFile, tt.wantText)
			}
		})
	}

	t.Run("missing dir", func(t *testing.T) {
		gotFile, gotText := findLicenseInDir(filepath.Join(t.TempDir(), "nope"))
		if gotFile != "" || gotText != "" {
			t.Errorf("findLicenseInDir(missing) = (%q, %q), want empty", gotFile, gotText)
		}
	})
}

func TestMainNpmPackagesErrors(t *testing.T) {
	t.Run("missing package.json", func(t *testing.T) {
		if _, err := mainNpmPackages(t.TempDir()); err == nil {
			t.Error("expected error for missing package.json")
		}
	})
	t.Run("invalid package.json", func(t *testing.T) {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{"package.json": "{not json"})
		if _, err := mainNpmPackages(dir); err == nil {
			t.Error("expected error for invalid package.json")
		}
	})
}

// harvestNpmLicenses only reads files under webDir, so it is tested against a
// synthetic node_modules tree.
func TestHarvestNpmLicenses(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"package.json": `{"dependencies": {
			"spdx-string": "1", "spdx-object": "1", "from-text": "1", "custom": "1",
			"no-license": "1", "missing": "1", "broken": "1", "@scope/pkg": "1",
			"bun-plugin-tailwind": "1", "@types/node": "1", " ": "1"
		}}`,
		"node_modules/spdx-string/package.json":         `{"version": "1.0.0", "license": "MIT", "repository": "owner/spdx-string"}`,
		"node_modules/spdx-object/package.json":         `{"version": "2.0.0", "license": {"type": "ISC"}, "repository": {"url": "git+https://github.com/o/obj.git"}}`,
		"node_modules/from-text/package.json":           `{"version": "3.0.0", "homepage": "https://from-text.dev"}`,
		"node_modules/from-text/LICENSE":                "Apache License\nVersion 2.0",
		"node_modules/custom/package.json":              `{"version": "4.0.0", "license": "Unknown"}`,
		"node_modules/custom/LICENSE":                   "Do what you want, but not that.",
		"node_modules/no-license/package.json":          `{"version": "5.0.0"}`,
		"node_modules/broken/package.json":              `{oops`,
		"node_modules/@scope/pkg/package.json":          `{"version": "6.0.0", "license": "BSD-2-Clause"}`,
		"node_modules/bun-plugin-tailwind/package.json": `{"version": "1.0.0", "license": "MIT"}`,
		"node_modules/@types/node/package.json":         `{"version": "1.0.0", "license": "MIT"}`,
	})

	got, err := harvestNpmLicenses(dir)
	if err != nil {
		t.Fatalf("harvestNpmLicenses: %v", err)
	}

	type row struct{ version, license, url string }
	want := map[string]row{
		"@scope/pkg":  {"6.0.0", "BSD-2-Clause", ""},
		"custom":      {"4.0.0", "Custom", ""},
		"from-text":   {"3.0.0", "Apache-2.0", "https://from-text.dev"},
		"no-license":  {"5.0.0", "Unknown", ""},
		"spdx-object": {"2.0.0", "ISC", "https://github.com/o/obj"},
		"spdx-string": {"1.0.0", "MIT", "https://github.com/owner/spdx-string"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d packages %+v, want %d (excluded, missing and broken packages skipped)", len(got), got, len(want))
	}
	for _, p := range got {
		w, ok := want[p.Name]
		if !ok {
			t.Errorf("unexpected package %q", p.Name)
			continue
		}
		if p.Ecosystem != "npm" || p.Version != w.version || p.License != w.license || p.URL != w.url {
			t.Errorf("%s = {eco %q, ver %q, lic %q, url %q}, want {npm, %q, %q, %q}",
				p.Name, p.Ecosystem, p.Version, p.License, p.URL, w.version, w.license, w.url)
		}
	}

	if _, err := harvestNpmLicenses(t.TempDir()); err == nil {
		t.Error("expected error when package.json is missing")
	}
}
