package csp_test

import (
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/csp"
)

func TestBuild(t *testing.T) {
	cfg := config.CSPConfig{
		Enabled:      true,
		DefaultSrc:   []string{"'self'"},
		ScriptSrc:    []string{"'self'"},
		StyleSrc:     []string{"'self'"},
		StyleSrcAttr: []string{"'unsafe-inline'"},
		ImgSrc:       []string{"'self'", "data:"},
	}
	scriptHash, styleHash := csp.WebAssetHashes()
	if !strings.HasPrefix(scriptHash, "'sha256-") || !strings.HasPrefix(styleHash, "'sha256-") {
		t.Fatalf("expected quoted sha256 web asset hashes, got %q %q", scriptHash, styleHash)
	}

	got := csp.Build(cfg, "https://cdn.example/x.js", "'sha384-abc'", "'self'", "")
	want := "default-src 'self'; " +
		"script-src 'self' " + scriptHash + " https://cdn.example/x.js 'sha384-abc'; " +
		"style-src 'self' " + styleHash + "; " +
		"style-src-attr 'unsafe-inline'; " +
		"img-src 'self' data:"
	if got != want {
		t.Errorf("Build =\n%q\nwant\n%q", got, want)
	}
	if len(cfg.ScriptSrc) != 1 {
		t.Errorf("Build must not modify the config slice: %v", cfg.ScriptSrc)
	}

	cfg.Enabled = false
	if got := csp.Build(cfg, "'sha384-abc'"); got != "" {
		t.Errorf("disabled policy = %q, want empty", got)
	}
}
