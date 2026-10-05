package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// newInternalSession builds a Server whose config can be adjusted before
// registration and connects an in-memory client session to it.
func newInternalSession(t *testing.T, mutate func(*config.Config)) (*Server, *store.Store, *sdk_mcp.ClientSession) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default(dir)
	if mutate != nil {
		mutate(cfg)
	}
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := New(service.New(cfg, dir, st, writer.New(writer.NewSuppressionCache(time.Second)), nil))
	clientT, serverT := sdk_mcp.NewInMemoryTransports()
	if _, err := srv.MCPServer().Connect(t.Context(), serverT, nil); err != nil {
		t.Fatal(err)
	}
	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "internal-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(t.Context(), clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return srv, st, session
}

// toolErrorText calls a tool and returns its error text, or "" on success.
func toolErrorText(t *testing.T, session *sdk_mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := session.CallTool(t.Context(), &sdk_mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	if !res.IsError {
		return ""
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdk_mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

func TestPriorityScore(t *testing.T) {
	tests := []struct {
		p    model.Priority
		want int
	}{
		{model.PriorityCritical, 4},
		{model.PriorityHigh, 3},
		{model.PriorityMedium, 2},
		{model.PriorityLow, 1},
		{"", 0},
		{"urgent", 0},
	}
	for _, tt := range tests {
		t.Run(string(tt.p), func(t *testing.T) {
			if got := priorityScore(tt.p); got != tt.want {
				t.Fatalf("priorityScore(%q) = %d, want %d", tt.p, got, tt.want)
			}
		})
	}
}

func TestResolveLimit(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero uses default", 0, 20},
		{"negative uses default", -5, 20},
		{"in range", 7, 7},
		{"at max", 100, 100},
		{"over max is capped", 101, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveLimit(tt.in); got != tt.want {
				t.Fatalf("resolveLimit(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestRequireID(t *testing.T) {
	if got, err := requireID("  abc ", "task"); err != nil || got != "abc" {
		t.Fatalf("requireID = %q, %v", got, err)
	}
	if _, err := requireID("   ", "task"); err == nil || err.Error() != "task id is required" {
		t.Fatalf("expected 'task id is required', got %v", err)
	}
}

func TestIsArchived(t *testing.T) {
	tests := []struct {
		name string
		ms   model.Milestone
		want bool
	}{
		{"closed", model.Milestone{Status: model.MilestoneStatusClosed}, true},
		{"open no tasks", model.Milestone{Status: model.MilestoneStatusOpen}, false},
		{"open all done", model.Milestone{Status: model.MilestoneStatusOpen, TotalTasks: 2, CompletedTasks: 2}, true},
		{"open partially done", model.Milestone{Status: model.MilestoneStatusOpen, TotalTasks: 2, CompletedTasks: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.IsArchived(tt.ms); got != tt.want {
				t.Fatalf("isArchived = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMutationsDisabled(t *testing.T) {
	_, _, session := newInternalSession(t, func(c *config.Config) { c.MCP.AllowMutations = false })

	tools := []struct {
		name string
		args map[string]any
	}{
		{"create_task", map[string]any{"title": "T", "summary": "s", "body": "b"}},
		{"update_task_status", map[string]any{"id": "x", "status": "ready"}},
		{"update_task_item", map[string]any{"id": "x", "index": 1, "completed": true}},
		{"update_task_content", map[string]any{"id": "x"}},
		{"complete_task", map[string]any{"id": "x", "summary": "s", "what_done": "w", "why_done": "y"}},
		{"delete_task", map[string]any{"id": "x"}},
		{"add_task_dependency", map[string]any{"id": "x", "dependency_id": "y"}},
		{"remove_task_dependency", map[string]any{"id": "x", "dependency_id": "y"}},
		{"add_task_note", map[string]any{"id": "x", "note": "n"}},
		{"set_task_target", map[string]any{"id": "x", "target_at": "2026-01-01"}},
		{"create_milestone", map[string]any{"title": "M", "summary": "s"}},
		{"update_milestone", map[string]any{"id": "x"}},
		{"delete_milestone", map[string]any{"id": "x"}},
		{"create_strategy", map[string]any{"title": "S", "tier": 1, "summary": "s"}},
		{"update_strategy", map[string]any{"id": "x"}},
		{"delete_strategy", map[string]any{"id": "x"}},
		{"create_glossary_term", map[string]any{"title": "G", "summary": "s"}},
		{"update_glossary_term", map[string]any{"id": "x"}},
		{"delete_glossary_term", map[string]any{"id": "x"}},
	}
	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			if got := toolErrorText(t, session, tt.name, tt.args); !strings.Contains(got, errMutationsDisabled.Error()) {
				t.Fatalf("%s: expected mutations-disabled error, got %q", tt.name, got)
			}
		})
	}
}

func TestStoreErrors(t *testing.T) {
	_, st, session := newInternalSession(t, nil)
	_ = st.Close()

	tools := []struct {
		name string
		args map[string]any
		want string
	}{
		{"search_tasks", map[string]any{"query": "x"}, "task search failed"},
		{"search_milestones", map[string]any{"query": "x"}, "milestone search failed"},
		{"search_strategies", map[string]any{"query": "x"}, "strategy search failed"},
		{"search_glossary", map[string]any{"query": "x"}, "glossary search failed"},
		{"search_all", map[string]any{"query": "x"}, "universal search failed"},
		{"lookup_glossary", map[string]any{"term": "x"}, "failed to list glossary terms"},
		{"get_task", map[string]any{"id": "x"}, "failed to get task"},
		{"get_milestone", map[string]any{"id": "x"}, "failed to get milestone"},
		{"get_strategy", map[string]any{"id": "x"}, "failed to get strategy"},
		{"list_tasks", map[string]any{}, ""},
		{"list_milestones", map[string]any{}, ""},
		{"list_strategies", map[string]any{}, ""},
		{"list_tags", map[string]any{}, ""},
		{"get_board_state", map[string]any{}, ""},
		{"list_task_items", map[string]any{"id": "x"}, ""},
	}
	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			got := toolErrorText(t, session, tt.name, tt.args)
			if got == "" {
				t.Fatalf("%s: expected an error with a closed store", tt.name)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("%s: error %q does not contain %q", tt.name, got, tt.want)
			}
		})
	}
}
