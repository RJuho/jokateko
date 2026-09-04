package web_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/RJuho/jokateko/web"
)

func TestEmbeddedDist(t *testing.T) {
	// 1. Verify pre-compressed gzip asset is embedded and non-empty
	gzData, err := web.GetGzipHTML()
	if err != nil {
		t.Fatalf("failed to read embedded dist/index.html.gz: %v", err)
	}
	if len(gzData) == 0 {
		t.Fatal("embedded dist/index.html.gz is empty")
	}

	// 2. Verify decompressed HTML getter
	htmlData, err := web.GetHTML()
	if err != nil {
		t.Fatalf("failed to get decompressed HTML: %v", err)
	}
	if len(htmlData) == 0 {
		t.Fatal("decompressed HTML is empty")
	}

	// 3. Verify that decompressed data from gzip matches GetHTML
	gzReader, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to read decompressed stream: %v", err)
	}

	if !bytes.Equal(decompressed, htmlData) {
		t.Fatal("GetHTML output does not match decompressed gzip stream")
	}

	// Verify expected HTML content markers
	if !bytes.Contains(htmlData, []byte("<!DOCTYPE html>")) {
		t.Fatal("decompressed HTML does not contain <!DOCTYPE html>")
	}
	if !bytes.Contains(htmlData, []byte("/* JOKATEKO_PAYLOAD_PLACEHOLDER */")) {
		t.Fatal("decompressed HTML does not contain JOKATEKO_PAYLOAD_PLACEHOLDER")
	}
}
