package version_test

import (
	"encoding/json"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/RJuho/jokateko/internal/version"
)

func TestGetLicensesRaw(t *testing.T) {
	raw := version.GetLicensesRaw()
	if len(raw) == 0 {
		t.Fatal("embedded licenses.json is empty")
	}
	if !json.Valid(raw) {
		t.Fatal("embedded licenses.json is not valid JSON")
	}
}

func TestGetLicenses(t *testing.T) {
	report := version.GetLicenses()

	if report.Project.Name != "Jokateko" {
		t.Errorf("project name = %q, want Jokateko", report.Project.Name)
	}
	if report.Project.License == "" || report.Project.Text == "" {
		t.Errorf("project license metadata incomplete: license=%q, text len=%d", report.Project.License, len(report.Project.Text))
	}
	if len(report.Packages) == 0 {
		t.Fatal("expected third-party packages in license report")
	}

	ecosystems := map[string]int{}
	for _, p := range report.Packages {
		if p.Name == "" || p.License == "" {
			t.Errorf("package entry missing name or license: %+v", p)
		}
		ecosystems[p.Ecosystem]++
	}
	for _, eco := range []string{"go", "npm"} {
		if ecosystems[eco] == 0 {
			t.Errorf("expected at least one %q package, got ecosystems %v", eco, ecosystems)
		}
	}

	// The parsed report must match a fresh decode of the raw bytes.
	var want version.LicenseReport
	if err := json.Unmarshal(version.GetLicensesRaw(), &want); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if len(want.Packages) != len(report.Packages) || want.Project != report.Project {
		t.Error("GetLicenses result differs from decoding GetLicensesRaw")
	}

	// Cached: a second call returns the same data.
	if again := version.GetLicenses(); len(again.Packages) != len(report.Packages) {
		t.Error("second GetLicenses call returned different data")
	}
}

// setVars overrides the link-time variables for the duration of the test.
func setVars(t *testing.T, v, commit, date, script, style string) {
	t.Helper()
	oldV, oldC, oldD, oldS, oldSt := version.Version, version.Commit, version.Date, version.ScriptHash, version.StyleHash
	t.Cleanup(func() {
		version.Version, version.Commit, version.Date, version.ScriptHash, version.StyleHash = oldV, oldC, oldD, oldS, oldSt
	})
	version.Version, version.Commit, version.Date, version.ScriptHash, version.StyleHash = v, commit, date, script, style
}

func TestGetInjectedValuesWin(t *testing.T) {
	setVars(t, "v1.2.3", "abc123", "2026-10-05T00:00:00Z", "sha256-script", "sha256-style")

	info := version.Get()
	want := version.Info{
		Version:    "v1.2.3",
		Commit:     "abc123",
		Date:       "2026-10-05T00:00:00Z",
		ScriptHash: "sha256-script",
		StyleHash:  "sha256-style",
		GoVersion:  runtime.Version(),
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
	}
	if info != want {
		t.Errorf("Get() = %+v, want %+v", info, want)
	}
	if s := info.String(); s != "jokateko v1.2.3 (commit: abc123, date: 2026-10-05T00:00:00Z, "+want.GoVersion+", "+want.Platform+")" {
		t.Errorf("String() = %q", s)
	}
}

func TestGetDevDefaultsFallback(t *testing.T) {
	setVars(t, "dev", "none", "unknown", "", "")

	info := version.Get()
	// Test binaries carry no VCS stamp and a "(devel)"/empty main version, so the
	// defaults may only be replaced by values read from runtime build info.
	if info.Version == "" || info.Commit == "" || info.Date == "" {
		t.Errorf("fallback produced empty fields: %+v", info)
	}
	if info.ScriptHash != "" || info.StyleHash != "" {
		t.Errorf("hashes must stay empty when not injected: %+v", info)
	}
}

// TestGetVCSFallback checks that VCS settings from runtime build info replace the
// "none"/"unknown" defaults. `go test` only stamps VCS info when run with
// -buildvcs=true, so without that flag this test skips.
func TestGetVCSFallback(t *testing.T) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no runtime build info")
	}
	settings := map[string]string{}
	for _, s := range bi.Settings {
		settings[s.Key] = s.Value
	}
	if settings["vcs.revision"] == "" {
		t.Skip("binary has no VCS stamp; run `go test -buildvcs=true` to exercise this path")
	}

	setVars(t, "dev", "none", "unknown", "", "")
	info := version.Get()
	if info.Commit != settings["vcs.revision"] {
		t.Errorf("Commit = %q, want vcs.revision %q", info.Commit, settings["vcs.revision"])
	}
	if info.Date != settings["vcs.time"] {
		t.Errorf("Date = %q, want vcs.time %q", info.Date, settings["vcs.time"])
	}

	// Injected values still win over build info.
	setVars(t, "dev", "injected", "injected-date", "", "")
	if info := version.Get(); info.Commit != "injected" || info.Date != "injected-date" {
		t.Errorf("injected values overridden by build info: %+v", info)
	}
}
