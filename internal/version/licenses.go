// Package version provides runtime version, commit, and open-source license information.
package version

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed licenses.json
var licensesJSON []byte

// ProjectLicense contains metadata and license text for Jokateko itself.
type ProjectLicense struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	URL     string `json:"url"`
	Text    string `json:"text"`
}

// PackageLicense contains metadata and license text for a third-party dependency.
type PackageLicense struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	License   string `json:"license"`
	Ecosystem string `json:"ecosystem"` // "go" or "npm"
	URL       string `json:"url,omitempty"`
	Text      string `json:"text,omitempty"`
}

// LicenseReport is the complete embedded dataset of project and third-party licenses.
type LicenseReport struct {
	Project  ProjectLicense   `json:"project"`
	Packages []PackageLicense `json:"packages"`
}

var (
	cachedLicenses LicenseReport
	licensesOnce   sync.Once
)

// GetLicenses returns the embedded open-source license report.
func GetLicenses() LicenseReport {
	licensesOnce.Do(func() {
		if len(licensesJSON) > 0 {
			_ = json.Unmarshal(licensesJSON, &cachedLicenses)
		}
	})
	return cachedLicenses
}

// GetLicensesRaw returns the raw embedded JSON bytes.
func GetLicensesRaw() []byte {
	return licensesJSON
}
