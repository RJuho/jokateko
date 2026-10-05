package exporter_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/exporter"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestSerializeSnapshot_Errors(t *testing.T) {
	t.Run("nil snapshot", func(t *testing.T) {
		if _, err := exporter.SerializeSnapshot(nil); err == nil || !strings.Contains(err.Error(), "cannot be nil") {
			t.Errorf("expected nil-snapshot error, got %v", err)
		}
	})

	t.Run("unencodable value", func(t *testing.T) {
		snap := &model.Snapshot{Milestones: []model.Milestone{{ID: "m", ProgressPercentage: math.NaN()}}}
		if _, err := exporter.SerializeSnapshot(snap); err == nil || !strings.Contains(err.Error(), "failed to serialize") {
			t.Errorf("expected serialize error for NaN, got %v", err)
		}
	})
}

func TestBuildSnapshot_NilConfigUsesDefaults(t *testing.T) {
	st := openStore(t)
	snap, err := exporter.BuildSnapshot(t.Context(), nil, st)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	def := config.Default("")
	if len(snap.Config.Board.Columns) != len(def.Board.Columns) {
		t.Errorf("expected %d default columns, got %d", len(def.Board.Columns), len(snap.Config.Board.Columns))
	}
	if snap.Tasks == nil || snap.Milestones == nil || snap.Strategies == nil || snap.Glossary == nil {
		t.Error("empty entity lists must be non-nil so they serialize as []")
	}
	if snap.Config.Build.Time == "" {
		t.Error("expected build time to be set")
	}
}

func TestBuildSnapshot_StoreErrors(t *testing.T) {
	tests := []struct {
		name       string
		breakStore func(t *testing.T, st *store.Store)
		wantErr    string
	}{
		{"closed store", func(t *testing.T, st *store.Store) { _ = st.Close() }, "failed to list tasks"},
		{"milestones table missing", dropTable("milestones"), "failed to list milestones"},
		{"strategies table missing", dropTable("strategies"), "failed to list strategies"},
		{"glossary table missing", dropTable("glossary"), "failed to list glossary terms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := openStore(t)
			tt.breakStore(t, st)
			_, err := exporter.BuildSnapshot(t.Context(), config.Default(""), st)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func dropTable(name string) func(t *testing.T, st *store.Store) {
	return func(t *testing.T, st *store.Store) {
		t.Helper()
		if _, err := st.DB().ExecContext(t.Context(), "DROP TABLE "+name); err != nil {
			t.Fatalf("drop %s: %v", name, err)
		}
	}
}

func TestExport_Errors(t *testing.T) {
	t.Run("snapshot build failure", func(t *testing.T) {
		st := openStore(t)
		_ = st.Close()
		out := filepath.Join(t.TempDir(), "index.html")
		_, err := exporter.Export(t.Context(), config.Default(""), st, out, exporter.MermaidNone)
		if err == nil || !strings.Contains(err.Error(), "failed to build snapshot") {
			t.Errorf("expected build error, got %v", err)
		}
		if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
			t.Error("no output file should be written on failure")
		}
	})

	t.Run("output directory cannot be created", func(t *testing.T) {
		st := openStore(t)
		dir := t.TempDir()
		blocker := filepath.Join(dir, "file")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := exporter.Export(t.Context(), config.Default(""), st, filepath.Join(blocker, "sub", "index.html"), exporter.MermaidNone)
		if err == nil || !strings.Contains(err.Error(), "failed to create export directory") {
			t.Errorf("expected mkdir error, got %v", err)
		}
	})

	t.Run("output path is a directory", func(t *testing.T) {
		st := openStore(t)
		out := t.TempDir()
		_, err := exporter.Export(t.Context(), config.Default(""), st, out, exporter.MermaidNone)
		if err == nil || !strings.Contains(err.Error(), "failed to write export file") {
			t.Errorf("expected write error, got %v", err)
		}
	})
}

func TestExport_DefaultOutPath(t *testing.T) {
	t.Chdir(t.TempDir())
	st := openStore(t)
	n, err := exporter.Export(t.Context(), config.Default(""), st, "", exporter.MermaidNone)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	info, err := os.Stat(filepath.Join("dist-kanban", "index.html"))
	if err != nil {
		t.Fatalf("default output not written: %v", err)
	}
	if info.Size() != n {
		t.Errorf("reported %d bytes, file has %d", n, info.Size())
	}
}
