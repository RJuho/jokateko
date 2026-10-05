// Package csp builds the Content-Security-Policy shared by the live server
// (HTTP header) and static exports (<meta http-equiv>).
package csp

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/version"
	"github.com/RJuho/jokateko/web"
)

// Build renders the policy from [server.security.csp], adding the hashes of the
// bundled inline script and style plus extraScriptSrc (e.g. the Mermaid runtime's
// SRI hash or CDN URL). It returns "" when the policy is disabled.
func Build(cfg config.CSPConfig, extraScriptSrc ...string) string {
	if !cfg.Enabled {
		return ""
	}

	scriptHash, styleHash := WebAssetHashes()

	scriptSrc := slices.Clone(cfg.ScriptSrc)
	for _, src := range append([]string{scriptHash}, extraScriptSrc...) {
		if src != "" && !slices.Contains(scriptSrc, src) {
			scriptSrc = append(scriptSrc, src)
		}
	}

	styleSrc := slices.Clone(cfg.StyleSrc)
	if styleHash != "" && !slices.Contains(styleSrc, styleHash) {
		styleSrc = append(styleSrc, styleHash)
	}

	var parts []string
	addDirective := func(name string, values []string) {
		if len(values) > 0 {
			parts = append(parts, fmt.Sprintf("%s %s", name, strings.Join(values, " ")))
		}
	}

	addDirective("default-src", cfg.DefaultSrc)
	addDirective("script-src", scriptSrc)
	addDirective("style-src", styleSrc)
	addDirective("style-src-elem", cfg.StyleSrcElem)
	addDirective("style-src-attr", cfg.StyleSrcAttr)
	addDirective("img-src", cfg.ImgSrc)
	addDirective("connect-src", cfg.ConnectSrc)
	addDirective("font-src", cfg.FontSrc)

	return strings.Join(parts, "; ")
}

// WebAssetHashes returns the CSP source expressions for the bundled inline script
// and style: the build-time ldflags values, or hashes computed from the embedded UI.
var WebAssetHashes = sync.OnceValues(func() (string, string) {
	scriptH := version.ScriptHash
	styleH := version.StyleHash

	if scriptH == "" || styleH == "" {
		htmlBytes, err := web.GetHTML()
		if err == nil {
			if scriptH == "" {
				scriptH = extractTagSHA256(htmlBytes, "script")
			}
			if styleH == "" {
				styleH = extractTagSHA256(htmlBytes, "style")
			}
		}
	}

	return quote(scriptH), quote(styleH)
})

func quote(src string) string {
	if src != "" && !strings.HasPrefix(src, "'") {
		return "'" + src + "'"
	}
	return src
}

func extractTagSHA256(html []byte, tag string) string {
	openTag := []byte("<" + tag + ">")
	closeTag := []byte("</" + tag + ">")

	_, rest, found := bytes.Cut(html, openTag)
	if !found {
		return ""
	}
	body, _, found := bytes.Cut(rest, closeTag)
	if !found {
		return ""
	}
	sum := sha256.Sum256(body)
	return fmt.Sprintf("'sha256-%s'", base64.StdEncoding.EncodeToString(sum[:]))
}
