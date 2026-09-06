// Package main generates an automated license report for all Go and Web dependencies.
// Run via 'go run ./cmd/genlicenses' or 'make generate'.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ProjectLicense struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	URL     string `json:"url"`
	Text    string `json:"text"`
}

type PackageLicense struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	License   string `json:"license"`
	Ecosystem string `json:"ecosystem"` // "go" or "npm"
	URL       string `json:"url,omitempty"`
	Text      string `json:"text,omitempty"`
}

type LicenseReport struct {
	Project  ProjectLicense   `json:"project"`
	Packages []PackageLicense `json:"packages"`
}

func main() {
	outGo := flag.String("out-go", "internal/version/licenses.json", "Output JSON path for Go embedding")
	outWeb := flag.String("out-web", "web/src/data/licenses.json", "Output JSON path for Web UI")
	licenseFile := flag.String("license", "LICENSE", "Path to project root LICENSE file")
	webDir := flag.String("web-dir", "web", "Directory containing web frontend package.json and node_modules")
	flag.Parse()

	// 1. Read root project license
	rootLicenseBytes, err := os.ReadFile(*licenseFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", *licenseFile, err)
		os.Exit(1)
	}

	report := LicenseReport{
		Project: ProjectLicense{
			Name:    "Jokateko",
			Version: "1.0.0",
			License: "MIT",
			URL:     "https://github.com/RJuho/jokateko",
			Text:    string(rootLicenseBytes),
		},
		Packages: make([]PackageLicense, 0),
	}

	// 2. Discover Go dependencies
	goPkgs, err := harvestGoLicenses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: harvesting go licenses failed: %v\n", err)
	} else {
		report.Packages = append(report.Packages, goPkgs...)
	}

	// 3. Discover Web / NPM dependencies
	npmPkgs, err := harvestNpmLicenses(*webDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: harvesting npm licenses failed: %v\n", err)
	} else {
		report.Packages = append(report.Packages, npmPkgs...)
	}

	// 4. Sort packages deterministically: ecosystem first ("go" then "npm"), then name
	slices.SortFunc(report.Packages, func(a, b PackageLicense) int {
		if a.Ecosystem != b.Ecosystem {
			return strings.Compare(a.Ecosystem, b.Ecosystem)
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	// 5. Serialize JSON with tab indentation matching project style
	data, err := json.MarshalIndent(report, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize licenses JSON: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	// 6. Write out-go
	if err := os.MkdirAll(filepath.Dir(*outGo), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create dir for %s: %v\n", *outGo, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outGo, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", *outGo, err)
		os.Exit(1)
	}

	// 7. Write out-web
	if err := os.MkdirAll(filepath.Dir(*outWeb), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create dir for %s: %v\n", *outWeb, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outWeb, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %v\n", *outWeb, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Harvested %d open-source licenses (%d Go modules, %d npm packages) -> %s, %s\n",
		len(report.Packages), len(goPkgs), len(npmPkgs), *outGo, *outWeb)
}

func harvestGoLicenses() ([]PackageLicense, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}", "./cmd/jokateko")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w (stderr: %s)", err, stderr.String())
	}

	seen := make(map[string]bool)
	var pkgs []PackageLicense

	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		modPath, version, dir := parts[0], parts[1], parts[2]
		if modPath == "github.com/RJuho/jokateko" || dir == "" || seen[modPath] {
			continue
		}
		seen[modPath] = true

		licFile, licText := findLicenseInDir(dir)
		spdx := detectSPDX(licText)
		if spdx == "Unknown" && licFile != "" {
			spdx = "Custom"
		}

		pkgs = append(pkgs, PackageLicense{
			Name:      modPath,
			Version:   version,
			License:   spdx,
			Ecosystem: "go",
			URL:       goModuleURL(modPath),
			Text:      licText,
		})
	}

	return pkgs, nil
}

func goModuleURL(modPath string) string {
	if strings.HasPrefix(modPath, "github.com/") {
		return "https://" + modPath
	}
	if strings.HasPrefix(modPath, "golang.org/x/") {
		sub := strings.TrimPrefix(modPath, "golang.org/x/")
		return "https://cs.opensource.google/go/x/" + sub
	}
	if strings.HasPrefix(modPath, "modernc.org/") {
		return "https://" + modPath
	}
	return "https://" + modPath
}

type npmPackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	License      any               `json:"license"`
	Repository   any               `json:"repository"`
	Homepage     string            `json:"homepage"`
	Dependencies map[string]string `json:"dependencies"`
}

func isExcludedNpmPackage(name string) bool {
	if strings.HasPrefix(name, "@types/") {
		return true
	}
	switch name {
	case "tailwindcss", "daisyui", "bun-plugin-tailwind":
		return true
	}
	if strings.HasPrefix(name, "@tailwindcss/") {
		return true
	}
	return false
}

func getBundledPackageNames(webDir string) ([]string, error) {
	// 1. Check if web/dist/bundled-packages.json exists
	bundledPath := filepath.Join(webDir, "dist", "bundled-packages.json")
	if data, err := os.ReadFile(bundledPath); err == nil {
		var pkgs []string
		if err := json.Unmarshal(data, &pkgs); err == nil && len(pkgs) > 0 {
			return pkgs, nil
		}
	}

	// 2. If Bun is installed, dynamically inspect Bun bundler metafile
	if _, err := exec.LookPath("bun"); err == nil {
		script := `const res = await Bun.build({ entrypoints: ['src/main.tsx'], target: 'browser', metafile: true });
const pkgs = new Set();
if (res.metafile) {
  for (const file of Object.keys(res.metafile.inputs)) {
    const m = file.match(/node_modules\/((?:@[^/]+\/)?[^/]+)/);
    if (m) pkgs.add(m[1]);
  }
}
console.log(JSON.stringify(Array.from(pkgs)));`
		cmd := exec.Command("bun", "-e", script)
		cmd.Dir = webDir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			var pkgs []string
			if err := json.Unmarshal(stdout.Bytes(), &pkgs); err == nil && len(pkgs) > 0 {
				return pkgs, nil
			}
		}
	}

	// 3. Fallback to web/package.json dependencies if bundle metadata or bun is unavailable
	rootPkgFile := filepath.Join(webDir, "package.json")
	rootData, err := os.ReadFile(rootPkgFile)
	if err != nil {
		return nil, fmt.Errorf("read root package.json: %w", err)
	}

	var rootPkg npmPackageJSON
	if err := json.Unmarshal(rootData, &rootPkg); err != nil {
		return nil, fmt.Errorf("unmarshal root package.json: %w", err)
	}

	var fallbackPkgs []string
	for dep := range rootPkg.Dependencies {
		fallbackPkgs = append(fallbackPkgs, dep)
	}
	return fallbackPkgs, nil
}

func harvestNpmLicenses(webDir string) ([]PackageLicense, error) {
	packageNames, err := getBundledPackageNames(webDir)
	if err != nil {
		return nil, fmt.Errorf("resolve bundled packages: %w", err)
	}

	nodeModulesDir := filepath.Join(webDir, "node_modules")
	seen := make(map[string]bool)
	var pkgs []PackageLicense

	for _, dep := range packageNames {
		dep = strings.TrimSpace(dep)
		if dep == "" || seen[dep] || isExcludedNpmPackage(dep) {
			continue
		}
		seen[dep] = true

		pkgDir := filepath.Join(nodeModulesDir, filepath.FromSlash(dep))
		pkgJSONFile := filepath.Join(pkgDir, "package.json")
		pkgData, err := os.ReadFile(pkgJSONFile)
		if err != nil {
			// Might be nested or bundled elsewhere
			continue
		}

		var p npmPackageJSON
		if err := json.Unmarshal(pkgData, &p); err != nil {
			continue
		}

		spdx := extractNpmLicense(p.License)
		licFile, licText := findLicenseInDir(pkgDir)
		if spdx == "" || spdx == "Unknown" {
			spdx = detectSPDX(licText)
		}
		if spdx == "Unknown" && licFile != "" {
			spdx = "Custom"
		}
		if spdx == "" {
			spdx = "Unknown"
		}

		url := extractNpmURL(p.Repository, p.Homepage)

		pkgs = append(pkgs, PackageLicense{
			Name:      dep,
			Version:   p.Version,
			License:   spdx,
			Ecosystem: "npm",
			URL:       url,
			Text:      licText,
		})
	}

	return pkgs, nil
}

func extractNpmLicense(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if t, ok := val["type"].(string); ok {
			return t
		}
	}
	return ""
}

func cleanURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	u = strings.TrimPrefix(u, "git+")
	u = strings.TrimPrefix(u, "git://")
	u = strings.TrimSuffix(u, ".git")

	if strings.HasPrefix(u, "ssh://git@github.com/") {
		u = "https://github.com/" + strings.TrimPrefix(u, "ssh://git@github.com/")
	} else if strings.HasPrefix(u, "git@github.com:") {
		u = "https://github.com/" + strings.TrimPrefix(u, "git@github.com:")
	} else if strings.HasPrefix(u, "github:") {
		u = "https://github.com/" + strings.TrimPrefix(u, "github:")
	} else if strings.HasPrefix(u, "github.com/") {
		u = "https://" + u
	} else if strings.HasPrefix(u, "http://") {
		u = "https://" + strings.TrimPrefix(u, "http://")
	} else if !strings.HasPrefix(u, "https://") {
		parts := strings.Split(u, "/")
		if len(parts) == 2 && !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") {
			u = "https://github.com/" + u
		} else if strings.Contains(u, ".") {
			u = "https://" + u
		}
	}
	return u
}

func extractNpmURL(repo any, homepage string) string {
	var repoURL string
	switch r := repo.(type) {
	case string:
		repoURL = r
	case map[string]any:
		if u, ok := r["url"].(string); ok {
			repoURL = u
		}
	}

	cleaned := cleanURL(repoURL)
	if cleaned != "" {
		return cleaned
	}
	return cleanURL(homepage)
}

func findLicenseInDir(dir string) (string, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}

	// Preferred candidate filenames in order
	candidates := []string{
		"license", "license.txt", "license.md",
		"licence", "licence.txt", "licence.md",
		"copying", "copying.txt", "unlicense",
	}

	for _, cand := range candidates {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.EqualFold(entry.Name(), cand) {
				full := filepath.Join(dir, entry.Name())
				data, err := os.ReadFile(full)
				if err == nil {
					return entry.Name(), string(data)
				}
			}
		}
	}

	// Fallback: any file whose prefix starts with "license" or "copying"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasPrefix(lower, "license") || strings.HasPrefix(lower, "copying") || strings.HasPrefix(lower, "unlicense") {
			full := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(full)
			if err == nil {
				return entry.Name(), string(data)
			}
		}
	}

	return "", ""
}

func detectSPDX(content string) string {
	if content == "" {
		return "Unknown"
	}
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "apache license") && strings.Contains(lower, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(lower, "mit license") || strings.Contains(lower, "permission is hereby granted, free of charge"):
		return "MIT"
	case strings.Contains(lower, "bsd 3-clause") || (strings.Contains(lower, "redistribution and use in source and binary forms") && strings.Contains(lower, "neither the name")):
		return "BSD-3-Clause"
	case strings.Contains(lower, "bsd 2-clause"):
		return "BSD-2-Clause"
	case strings.Contains(lower, "mozilla public license") || strings.Contains(lower, "mpl 2.0"):
		return "MPL-2.0"
	case strings.Contains(lower, "isc license") || strings.Contains(lower, "permission to use, copy, modify, and/or distribute this software"):
		return "ISC"
	case strings.Contains(lower, "this is free and unencumbered software") || strings.Contains(lower, "unlicense"):
		return "Unlicense"
	case strings.Contains(lower, "creative commons zero") || strings.Contains(lower, "cc0 1.0"):
		return "CC0-1.0"
	default:
		return "Unknown"
	}
}
