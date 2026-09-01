package web_test

import (
	"io/fs"
	"testing"

	"github.com/RJuho/jokateko/web"
)

func TestEmbeddedDist(t *testing.T) {
	data, err := fs.ReadFile(web.Dist, "dist/index.html")
	if err != nil {
		t.Fatalf("failed to read embedded dist/index.html: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("embedded dist/index.html is empty")
	}
}
