// Package version provides runtime version, commit, and build date information.
// Values can be set at link time via -ldflags during release builds.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

var (
	// Version is the semantic version of Jokateko, injected at build time (e.g. "v1.1.1").
	Version = "dev"

	// Commit is the Git commit SHA from which the binary was built.
	Commit = "none"

	// Date is the UTC build timestamp in ISO 8601 / RFC 3339 format.
	Date = "unknown"

	// ScriptHash is the CSP sha256 hash of the bundled inline script (e.g. "sha256-..."),
	// injected at build time via -ldflags.
	ScriptHash = ""

	// StyleHash is the CSP sha256 hash of the bundled inline stylesheet (e.g. "sha256-..."),
	// injected at build time via -ldflags.
	StyleHash = ""
)

// Info holds structured build metadata for diagnostics and API responses.
type Info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	Date       string `json:"date"`
	ScriptHash string `json:"script_hash,omitempty"`
	StyleHash  string `json:"style_hash,omitempty"`
	GoVersion  string `json:"go_version"`
	Platform   string `json:"platform"`
}

// Get returns the resolved build and version information.
// When running an unlinked development binary, it attempts to populate
// missing fields from Go's runtime build info.
func Get() Info {
	info := Info{
		Version:    Version,
		Commit:     Commit,
		Date:       Date,
		ScriptHash: ScriptHash,
		StyleHash:  StyleHash,
		GoVersion:  runtime.Version(),
		Platform:   fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// Fallback to runtime/debug.ReadBuildInfo() if not injected via -ldflags.
	if bi, ok := debug.ReadBuildInfo(); ok {
		if info.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}

		for _, setting := range bi.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "none" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.Date == "unknown" {
					info.Date = setting.Value
				}
			}
		}
	}

	return info
}

// String returns a human-readable representation of the version info.
func (i Info) String() string {
	return fmt.Sprintf("jokateko %s (commit: %s, date: %s, %s, %s)",
		i.Version, i.Commit, i.Date, i.GoVersion, i.Platform)
}
