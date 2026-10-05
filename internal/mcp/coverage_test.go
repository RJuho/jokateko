package mcp_test

import (
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	internalmcp "github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func expectToolError(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", contains)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("error %q does not contain %q", err.Error(), contains)
	}
}

func mustCreateTask(t *testing.T, session *mcp.ClientSession, in internalmcp.CreateTaskInput) internalmcp.TaskDetail {
	t.Helper()
	if in.Body == "" {
		in.Body = "Body"
	}
	task, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", in)
	if err != nil {
		t.Fatalf("create_task: %v", err)
	}
	return task
}

func TestMCP_SearchEntityTools(t *testing.T) {
	_, _, _, session := setupTestMCP(t)

	if _, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{
		Title: "Quokka Release", Summary: "Ship the quokka features", Tags: []string{"feature"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
		Title: "Quokka Invariants", Tier: 1, Summary: "Quokka rules for the backend", Tags: []string{"backend"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
		Title: "Quokka", Summary: "A small marsupial used as a codename", Tags: []string{"api"},
	}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		tool    string
		query   string
		tag     string
		limit   int
		wantHit bool
	}{
		{"search_milestones", "quokka", "", 0, true},
		{"search_milestones", "wombat", "", 0, false},
		{"search_milestones", "quokka", "feature", 5, true},
		{"search_milestones", "quokka", "backend", 0, false},
		{"search_strategies", "quokka", "", -1, true},
		{"search_strategies", "wombat", "", 0, false},
		{"search_strategies", "quokka", "backend", 500, true},
		{"search_strategies", "quokka", "api", 0, false},
		{"search_glossary", "marsupial", "", 0, true},
		{"search_glossary", "wombat", "", 0, false},
		{"search_glossary", "quokka", "api", 1, true},
		{"search_glossary", "quokka", "feature", 0, false},
	}
	for _, tt := range tests {
		name := tt.tool + "/" + tt.query
		if tt.tag != "" {
			name += "/tag=" + tt.tag
		}
		t.Run(name, func(t *testing.T) {
			res, err := callToolJSON[[]model.SearchResult](t, session, tt.tool, internalmcp.SearchInput{Query: tt.query, Tag: tt.tag, Limit: tt.limit})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(res) > 0; got != tt.wantHit {
				t.Fatalf("hit = %v, want %v (results %+v)", got, tt.wantHit, res)
			}
		})
	}
}

func TestMCP_SetTaskTarget(t *testing.T) {
	_, _, _, session := setupTestMCP(t)
	task := mustCreateTask(t, session, internalmcp.CreateTaskInput{Title: "Target me", Summary: "s"})

	tests := []struct {
		name      string
		id        string
		target    string
		want      string
		wantErr   string
		wantMsg   string
		inFile    string
		notInFile string
	}{
		{name: "rfc3339", id: task.ID, target: "2026-12-24T18:00:00Z", want: "2026-12-24T18:00:00Z", wantMsg: "updated to", inFile: "target_at = '2026-12-24T18:00:00Z'"},
		{name: "date only", id: task.ID, target: "2027-01-15", want: "2027-01-15T00:00:00Z", wantMsg: "updated to", inFile: "target_at = '2027-01-15T00:00:00Z'"},
		{name: "invalid date", id: task.ID, target: "15/01/2027", wantErr: "invalid target_at format"},
		{name: "clear", id: task.ID, target: "", wantMsg: "cleared", notInFile: "target_at"},
		{name: "unknown id", id: "missing", target: "2027-01-15", wantErr: "not found"},
		{name: "empty id", id: " ", target: "2027-01-15", wantErr: "task id is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := callToolJSON[internalmcp.SetTaskTargetOutput](t, session, "set_task_target", internalmcp.SetTaskTargetInput{ID: tt.id, TargetAt: tt.target})
			if tt.wantErr != "" {
				expectToolError(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !out.Success || out.TargetAt != tt.want || !strings.Contains(out.Message, tt.wantMsg) {
				t.Fatalf("unexpected output %+v", out)
			}
			data, err := os.ReadFile(task.FilePath)
			if err != nil {
				t.Fatal(err)
			}
			if tt.inFile != "" && !strings.Contains(string(data), tt.inFile) {
				t.Fatalf("file missing %q:\n%s", tt.inFile, data)
			}
			if tt.notInFile != "" && strings.Contains(string(data), tt.notInFile) {
				t.Fatalf("file unexpectedly contains %q:\n%s", tt.notInFile, data)
			}
		})
	}
}

func TestMCP_UpdateTaskContentErrors(t *testing.T) {
	_, _, _, session := setupTestMCP(t)
	task := mustCreateTask(t, session, internalmcp.CreateTaskInput{Title: "Content", Summary: "s", Body: "Original body"})

	closed, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{Title: "Closed MS", Summary: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "update_milestone", internalmcp.UpdateMilestoneInput{ID: closed.ID, Status: new("closed")}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		in      internalmcp.UpdateTaskContentInput
		wantErr string
	}{
		{"empty id", internalmcp.UpdateTaskContentInput{ID: ""}, "task id is required"},
		{"disallowed tag", internalmcp.UpdateTaskContentInput{ID: task.ID, Tags: &[]string{"nope"}}, "not permitted"},
		{"closed milestone", internalmcp.UpdateTaskContentInput{ID: task.ID, Milestone: new(closed.ID)}, "reopen_milestone"},
		{"invalid target", internalmcp.UpdateTaskContentInput{ID: task.ID, TargetAt: new("someday")}, "invalid target_at format"},
		{"unknown id", internalmcp.UpdateTaskContentInput{ID: "missing", Title: new("x")}, "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", tt.in)
			expectToolError(t, err, tt.wantErr)
		})
	}

	t.Run("body locked outside editable states", func(t *testing.T) {
		if _, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_status", internalmcp.UpdateTaskStatusInput{ID: task.ID, Status: "ready"}); err != nil {
			t.Fatal(err)
		}
		_, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{ID: task.ID, Body: new("Rewritten")})
		expectToolError(t, err, "add_task_note")
		data, _ := os.ReadFile(task.FilePath)
		if !strings.Contains(string(data), "Original body") {
			t.Fatalf("body changed despite rejection:\n%s", data)
		}
	})

	t.Run("reopen milestone and all fields", func(t *testing.T) {
		got, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
			ID:              task.ID,
			Title:           new("New title"),
			Summary:         new("New summary"),
			Priority:        new("critical"),
			Milestone:       new(closed.ID),
			ReopenMilestone: true,
			Tags:            &[]string{"api"},
			Dependencies:    &[]string{},
			TargetAt:        new("2027-02-01"),
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Title != "New title" || got.Summary != "New summary" || got.Priority != model.PriorityCritical ||
			got.Milestone != closed.ID || !slices.Equal(got.Tags, []string{"api"}) || got.TargetAt != "2027-02-01T00:00:00Z" {
			t.Fatalf("unexpected task %+v", got)
		}
	})
}

func TestMCP_TaskNotFoundPaths(t *testing.T) {
	_, _, _, session := setupTestMCP(t)

	t.Run("get_task", func(t *testing.T) {
		_, err := callToolJSON[internalmcp.TaskDetail](t, session, "get_task", internalmcp.GetTaskInput{ID: "missing"})
		expectToolError(t, err, `task "missing" not found`)
		_, err = callToolJSON[internalmcp.TaskDetail](t, session, "get_task", internalmcp.GetTaskInput{ID: " "})
		expectToolError(t, err, "task id is required")
	})
	t.Run("add_task_note", func(t *testing.T) {
		_, err := callToolJSON[internalmcp.AddTaskNoteOutput](t, session, "add_task_note", internalmcp.AddTaskNoteInput{ID: "missing", Note: "n"})
		expectToolError(t, err, "not found")
		_, err = callToolJSON[internalmcp.AddTaskNoteOutput](t, session, "add_task_note", internalmcp.AddTaskNoteInput{ID: "", Note: "n"})
		expectToolError(t, err, "task id is required")
	})
}

func TestMCP_UpdateMilestoneBranches(t *testing.T) {
	_, _, _, session := setupTestMCP(t)
	ms, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{Title: "Beta", Summary: "beta release"})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		in      internalmcp.UpdateMilestoneInput
		wantErr string
		check   func(t *testing.T, got internalmcp.MilestoneDetail)
	}{
		{name: "close", in: internalmcp.UpdateMilestoneInput{ID: ms.ID, Status: new("closed")}, check: func(t *testing.T, got internalmcp.MilestoneDetail) {
			if got.Status != model.MilestoneStatusClosed || !got.IsArchived {
				t.Fatalf("expected closed+archived, got %+v", got)
			}
		}},
		{name: "reopen", in: internalmcp.UpdateMilestoneInput{ID: ms.ID, Status: new("OPEN")}, check: func(t *testing.T, got internalmcp.MilestoneDetail) {
			if got.Status != model.MilestoneStatusOpen || got.IsArchived {
				t.Fatalf("expected open, got %+v", got)
			}
		}},
		{name: "timeframe and fields", in: internalmcp.UpdateMilestoneInput{
			ID: ms.ID, TargetDate: new(" 2027-03-01 "), Summary: new("new summary"), Tags: &[]string{"feature"}, Body: new("Roadmap"),
		}, check: func(t *testing.T, got internalmcp.MilestoneDetail) {
			if got.TargetDate != "2027-03-01" || got.Summary != "new summary" || !slices.Equal(got.Tags, []string{"feature"}) || got.Body != "Roadmap" {
				t.Fatalf("unexpected milestone %+v", got)
			}
		}},
		{name: "blank summary keeps old", in: internalmcp.UpdateMilestoneInput{ID: ms.ID, Summary: new("  ")}, check: func(t *testing.T, got internalmcp.MilestoneDetail) {
			if got.Summary != "new summary" {
				t.Fatalf("summary = %q, want unchanged", got.Summary)
			}
		}},
		{name: "invalid status", in: internalmcp.UpdateMilestoneInput{ID: ms.ID, Status: new("archived")}, wantErr: "invalid status"},
		{name: "disallowed tag", in: internalmcp.UpdateMilestoneInput{ID: ms.ID, Tags: &[]string{"nope"}}, wantErr: "not permitted"},
		{name: "empty id", in: internalmcp.UpdateMilestoneInput{ID: ""}, wantErr: "milestone id is required"},
		{name: "unknown id", in: internalmcp.UpdateMilestoneInput{ID: "missing"}, wantErr: "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "update_milestone", tt.in)
			if tt.wantErr != "" {
				expectToolError(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, got)
			data, err := os.ReadFile(ms.FilePath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "status = '"+string(got.Status)+"'") {
				t.Fatalf("file status mismatch:\n%s", data)
			}
		})
	}

	t.Run("get_milestone not found", func(t *testing.T) {
		_, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "get_milestone", internalmcp.GetMilestoneInput{ID: "missing"})
		expectToolError(t, err, "not found")
		_, err = callToolJSON[internalmcp.MilestoneDetail](t, session, "get_milestone", internalmcp.GetMilestoneInput{ID: " "})
		expectToolError(t, err, "milestone id is required")
	})
	t.Run("create_milestone guards", func(t *testing.T) {
		_, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{Title: "X"})
		expectToolError(t, err, "summary is required")
		_, err = callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{Title: "X", Summary: "s", Tags: []string{"nope"}})
		expectToolError(t, err, "not permitted")
		_, err = callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", internalmcp.CreateMilestoneInput{Title: " ", Summary: "s"})
		expectToolError(t, err, "title is required")
	})
	t.Run("delete_milestone guards", func(t *testing.T) {
		_, err := callToolJSON[internalmcp.DeleteEntityOutput](t, session, "delete_milestone", internalmcp.DeleteMilestoneInput{ID: " "})
		expectToolError(t, err, "milestone id is required")
		_, err = callToolJSON[internalmcp.DeleteEntityOutput](t, session, "delete_milestone", internalmcp.DeleteMilestoneInput{ID: "missing"})
		expectToolError(t, err, "not found")
	})
}

func TestMCP_StrategyGlossaryNotFound(t *testing.T) {
	_, _, _, session := setupTestMCP(t)

	tests := []struct {
		name    string
		tool    string
		args    any
		wantErr string
	}{
		{"delete strategy unknown", "delete_strategy", internalmcp.DeleteStrategyInput{ID: "missing"}, "not found"},
		{"delete strategy empty", "delete_strategy", internalmcp.DeleteStrategyInput{ID: ""}, "strategy id is required"},
		{"get strategy unknown", "get_strategy", internalmcp.GetStrategyInput{ID: "missing"}, "not found"},
		{"get strategy empty", "get_strategy", internalmcp.GetStrategyInput{ID: " "}, "strategy id is required"},
		{"update strategy empty", "update_strategy", internalmcp.UpdateStrategyInput{ID: ""}, "strategy id is required"},
		{"update strategy unknown", "update_strategy", internalmcp.UpdateStrategyInput{ID: "missing", Summary: "x"}, "not found"},
		{"delete glossary unknown", "delete_glossary_term", internalmcp.DeleteGlossaryTermInput{ID: "missing"}, "not found"},
		{"delete glossary empty", "delete_glossary_term", internalmcp.DeleteGlossaryTermInput{ID: ""}, "glossary term id is required"},
		{"update glossary empty", "update_glossary_term", internalmcp.UpdateGlossaryTermInput{ID: ""}, "term id is required"},
		{"update glossary unknown", "update_glossary_term", internalmcp.UpdateGlossaryTermInput{ID: "missing", Summary: "x"}, "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callToolJSON[map[string]any](t, session, tt.tool, tt.args)
			expectToolError(t, err, tt.wantErr)
		})
	}

	t.Run("lookup_glossary miss returns empty", func(t *testing.T) {
		res, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{Term: "nonexistent-term"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 0 {
			t.Fatalf("expected no results, got %+v", res)
		}
	})

	t.Run("lookup_glossary falls back to full-text search", func(t *testing.T) {
		if _, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
			Title: "Spec First", Summary: "Write the specification before implementation",
		}); err != nil {
			t.Fatal(err)
		}
		res, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{Term: "specification"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 || res[0].ID != "spec-first" {
			t.Fatalf("expected FTS fallback to find spec-first, got %+v", res)
		}
	})
}

func TestMCP_HTTPHandlerRoundTrip(t *testing.T) {
	srv, _, _, _ := setupTestMCP(t)
	ts := httptest.NewServer(srv.HTTPHandler())
	t.Cleanup(ts.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "http-test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: ts.URL, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("initialize over HTTP: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	if res := session.InitializeResult(); res == nil || res.ServerInfo == nil || res.ServerInfo.Name != "jokateko" {
		t.Fatalf("unexpected initialize result %+v", res)
	}

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("tools/list over HTTP: %v", err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	for _, want := range []string{"create_task", "get_task", "search_all", "set_task_target", "update_milestone"} {
		if !slices.Contains(names, want) {
			t.Errorf("tools/list missing %q (got %v)", want, names)
		}
	}
}

func TestMCP_ResourceStrategyTiers(t *testing.T) {
	_, _, _, session := setupTestMCP(t)
	res, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: "jokateko://strategies/tiers"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(res.Contents))
	}
	c := res.Contents[0]
	if c.URI != "jokateko://strategies/tiers" || c.MIMEType != "text/markdown" {
		t.Fatalf("unexpected content metadata %+v", c)
	}
	for _, want := range []string{
		"# Architectural Strategy Tiers",
		"## Tier 1: Core Architecture & Tech Stack (`tier = 1`)",
		"## Tier 3: Code Conventions & UI Standards (`tier = 3`)",
		"System-wide non-negotiables",
	} {
		if !strings.Contains(c.Text, want) {
			t.Errorf("tiers resource missing %q:\n%s", want, c.Text)
		}
	}
}
