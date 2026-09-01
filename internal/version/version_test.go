package version_test

import (
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/version"
)

func TestGet(t *testing.T) {
	info := version.Get()

	if info.Version == "" {
		t.Error("expected non-empty Version")
	}

	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}

	if info.Platform == "" {
		t.Error("expected non-empty Platform")
	}

	str := info.String()
	if !strings.Contains(str, "jokateko") {
		t.Errorf("expected string representation to contain 'jokateko', got: %s", str)
	}
}
