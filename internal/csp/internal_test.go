package csp

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestExtractTagSHA256(t *testing.T) {
	sum := sha256.Sum256([]byte("body{}"))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"

	tests := []struct {
		name, html, tag, want string
	}{
		{"found", "<html><style>body{}</style></html>", "style", want},
		{"first occurrence wins", "<style>body{}</style><style>other</style>", "style", want},
		{"missing open tag", "<html></html>", "style", ""},
		{"missing close tag", "<style>body{}", "style", ""},
		{"tag with attributes is not matched", `<script type="module">x</script>`, "script", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractTagSHA256([]byte(tc.html), tc.tag); got != tc.want {
				t.Errorf("extractTagSHA256 = %q, want %q", got, tc.want)
			}
		})
	}
}
