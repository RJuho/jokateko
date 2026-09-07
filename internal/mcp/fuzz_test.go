package mcp

import (
	"bytes"
	"context"
	"testing"
	"time"
	"io"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type nopWriteCloser struct{}

func (nopWriteCloser) Write(p []byte) (n int, err error) {
	return len(p), nil
}
func (nopWriteCloser) Close() error {
	return nil
}

func FuzzMCPServer(f *testing.F) {
	f.Add([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_task","arguments":{"id":"123"}}}`))
	f.Add([]byte(`malformed json`))

	f.Fuzz(func(t *testing.T, payload []byte) {
		st, err := store.OpenMemory()
		if err != nil {
			t.Fatal(err)
		}
		defer st.Close()

		sc := writer.NewSuppressionCache(time.Second)
		wr := writer.New(sc)
		
		cfg := &config.Config{}
		srv := New(cfg, t.TempDir(), st, wr)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		transport := &sdk_mcp.IOTransport{
			Reader: io.NopCloser(bytes.NewReader(payload)),
			Writer: nopWriteCloser{},
		}
		
		_ = srv.MCPServer().Run(ctx, transport)
	})
}
