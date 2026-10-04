package exporter_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/exporter"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/web"
)

func TestBuildSnapshot(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	ctx := t.Context()
	cfg := config.Default(t.TempDir())
	cfg.Project.Name = "Test Project"

	// Seed task
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-1",
		Title:    "First Task",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Summary:  "Summary 1",
	})

	// Seed milestone
	_ = st.UpsertMilestone(ctx, model.Milestone{
		ID:    "ms-1",
		Title: "Milestone 1",
	})

	// Seed strategy
	_ = st.UpsertStrategy(ctx, model.Strategy{
		ID:    "strat-1",
		Title: "Strategy 1",
		Tier:  model.TierCore,
	})

	// Seed glossary
	_ = st.UpsertGlossaryTerm(ctx, model.GlossaryTerm{
		ID:    "term-1",
		Title: "Term 1",
	})

	snap, err := exporter.BuildSnapshot(ctx, cfg, st)
	if err != nil {
		t.Fatalf("BuildSnapshot failed: %v", err)
	}

	if snap.Config.Project.Name != "Test Project" {
		t.Errorf("expected project name 'Test Project', got %q", snap.Config.Project.Name)
	}
	if len(snap.Tasks) != 1 || snap.Tasks[0].ID != "task-1" {
		t.Errorf("expected 1 task, got %+v", snap.Tasks)
	}
	if len(snap.Milestones) != 1 || snap.Milestones[0].ID != "ms-1" {
		t.Errorf("expected 1 milestone, got %+v", snap.Milestones)
	}
	if len(snap.Strategies) != 1 || snap.Strategies[0].ID != "strat-1" {
		t.Errorf("expected 1 strategy, got %+v", snap.Strategies)
	}
	if len(snap.Glossary) != 1 || snap.Glossary[0].ID != "term-1" {
		t.Errorf("expected 1 glossary term, got %+v", snap.Glossary)
	}
}

func TestSerializeSnapshot_HTMLSafe(t *testing.T) {
	snap := &model.Snapshot{
		Tasks: []model.Task{
			{
				ID:      "dangerous-task",
				Title:   "<script>alert('xss')</script>",
				Summary: "Tags like <b> & <i> must be escaped",
			},
		},
	}

	data, err := exporter.SerializeSnapshot(snap)
	if err != nil {
		t.Fatalf("SerializeSnapshot failed: %v", err)
	}

	raw := string(data)
	// Must not contain literal </script>
	if strings.Contains(raw, "</script>") {
		t.Errorf("JSON output contains unescaped </script>: %s", raw)
	}
	if !strings.Contains(raw, `\u003cscript\u003e`) {
		t.Errorf("expected HTML-escaped opening script tag, got %s", raw)
	}

	// Must unmarshal cleanly back to original struct
	var roundTrip model.Snapshot
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if roundTrip.Tasks[0].Title != "<script>alert('xss')</script>" {
		t.Errorf("round-trip title mismatch: %q", roundTrip.Tasks[0].Title)
	}
}

func TestInjectSnapshot(t *testing.T) {
	template := []byte(`<html><head><script id="jokateko-data">/* JOKATEKO_PAYLOAD_PLACEHOLDER */</script></head><body></body></html>`)
	payload := []byte(`{"project":"jokateko"}`)

	// 1. Success case
	injected, err := exporter.InjectSnapshot(template, payload)
	if err != nil {
		t.Fatalf("InjectSnapshot failed: %v", err)
	}
	expected := `<html><head><script id="jokateko-data">{"project":"jokateko"}</script></head><body></body></html>`
	if string(injected) != expected {
		t.Errorf("unexpected injected HTML:\n%s", string(injected))
	}

	// 2. Missing placeholder
	noPlaceholder := []byte(`<html><head></head><body>No placeholder here</body></html>`)
	_, err = exporter.InjectSnapshot(noPlaceholder, payload)
	if err == nil {
		t.Fatal("expected error when placeholder is missing")
	}
}

func TestExport_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "dist", "index.html")

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	ctx := t.Context()
	cfg := config.Default(dir)
	cfg.Project.Name = "Offline Board"

	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-offline",
		Title:    "Offline Task",
		Status:   "done",
		Priority: model.PriorityMedium,
		Summary:  "Snapshot exported task",
	})

	bytesWritten, err := exporter.Export(ctx, cfg, st, outPath, exporter.MermaidCDN)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if bytesWritten <= 0 {
		t.Errorf("expected positive bytes written, got %d", bytesWritten)
	}

	// Verify file on disk
	fileContent, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	if !bytes.Contains(fileContent, []byte("task-offline")) {
		t.Errorf("expected exported file to contain task-offline")
	}
	if bytes.Contains(fileContent, []byte(exporter.PayloadPlaceholder)) {
		t.Errorf("exported file should no longer contain placeholder")
	}
}

func TestParseMermaidMode(t *testing.T) {
	for in, want := range map[string]exporter.MermaidMode{
		"cdn":     exporter.MermaidCDN,
		"bundled": exporter.MermaidBundled,
		" NONE ":  exporter.MermaidNone,
	} {
		got, err := exporter.ParseMermaidMode(in)
		if err != nil || got != want {
			t.Errorf("ParseMermaidMode(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := exporter.ParseMermaidMode("inline"); err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestInjectMermaid(t *testing.T) {
	template := []byte("<html><head><title>x</title></head><body><div id=\"app\"></div></body></html>")
	runtime := []byte("globalThis.mermaid={};")

	for _, mode := range []exporter.MermaidMode{exporter.MermaidCDN, exporter.MermaidNone} {
		out, err := exporter.InjectMermaid(template, mode, runtime)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if !bytes.Contains(out, []byte(`<meta name="jokateko-mermaid" content="`+string(mode)+`" />`+"\n</head>")) {
			t.Errorf("%s: expected mode meta before </head>, got %s", mode, out)
		}
		if bytes.Contains(out, runtime) {
			t.Errorf("%s: runtime must not be inlined", mode)
		}
	}

	out, err := exporter.InjectMermaid(template, exporter.MermaidBundled, runtime)
	if err != nil {
		t.Fatalf("bundled: %v", err)
	}
	wantBlock := `<script type="text/plain" id="jokateko-mermaid-src">globalThis.mermaid={};</script>` + "\n</body>"
	if !bytes.Contains(out, []byte(wantBlock)) {
		t.Errorf("bundled: expected inert runtime block before </body>, got %s", out)
	}

	// A runtime that would terminate the inert block early must be rejected
	if _, err := exporter.InjectMermaid(template, exporter.MermaidBundled, []byte("a='</SCRIPT>'")); err == nil {
		t.Error("expected error for runtime containing </script")
	}
}

func TestExport_MermaidModes(t *testing.T) {
	dir := t.TempDir()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()
	cfg := config.Default(dir)

	runtime, err := web.GetMermaidRuntime()
	if err != nil {
		t.Fatalf("embedded mermaid runtime: %v", err)
	}
	runtimeJS, err := web.GetMermaidJS()
	if err != nil {
		t.Fatalf("embedded mermaid js: %v", err)
	}

	sizes := map[exporter.MermaidMode]int64{}
	for _, mode := range []exporter.MermaidMode{exporter.MermaidCDN, exporter.MermaidBundled, exporter.MermaidNone} {
		outPath := filepath.Join(dir, string(mode)+".html")
		size, err := exporter.Export(t.Context(), cfg, st, outPath, mode)
		if err != nil {
			t.Fatalf("%s: Export failed: %v", mode, err)
		}
		sizes[mode] = size

		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if !bytes.Contains(content, []byte(`content="`+string(mode)+`"`)) {
			t.Errorf("%s: missing jokateko-mermaid meta", mode)
		}
		if got := bytes.Contains(content, runtimeJS); got != (mode == exporter.MermaidBundled) {
			t.Errorf("%s: runtime inlined = %v", mode, got)
		}
	}

	if sizes[exporter.MermaidBundled] < sizes[exporter.MermaidCDN]+int64(len(runtimeJS)) {
		t.Errorf("bundled export (%d) should be larger than cdn export (%d) by the runtime", sizes[exporter.MermaidBundled], sizes[exporter.MermaidCDN])
	}
	if !strings.Contains(runtime.CDNURL(), "mermaid@"+runtime.Version+"/dist/mermaid.min.js") {
		t.Errorf("unexpected CDN URL %q", runtime.CDNURL())
	}
}
