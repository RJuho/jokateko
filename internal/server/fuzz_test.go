package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

func FuzzAPIHandlers(f *testing.F) {
	// Add some seed payloads
	f.Add("POST", "/api/tasks", []byte(`{"title":"New Task"}`))
	f.Add("PATCH", "/api/tasks/123/status", []byte(`{"status":"in_progress"}`))
	f.Add("GET", "/api/tasks", []byte{})

	f.Fuzz(func(t *testing.T, method, path string, body []byte) {
		// Basic validation to prevent routing random non-matching strings
		if len(method) == 0 || len(path) == 0 || path[0] != '/' {
			return
		}

		st, err := store.OpenMemory()
		if err != nil {
			t.Fatal(err)
		}
		defer st.Close()

		sc := writer.NewSuppressionCache(time.Second)
		wr := writer.New(sc)
		
		cfg := &config.Config{}
		cfg.Board.Columns = []config.ColumnConfig{{ID: "backlog"}}
		cfg.Tags.Allowed = []string{"bug"}

		srv := New(cfg, t.TempDir(), st, wr, nil)

		req, err := http.NewRequest(method, path, bytes.NewReader(body))
		if err != nil {
			return
		}
		w := httptest.NewRecorder()

		srv.handler.ServeHTTP(w, req)
	})
}
