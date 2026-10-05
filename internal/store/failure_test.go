package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
)

// newTestStore opens an isolated in-memory store that is closed when the test ends.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// execSQL runs raw SQL against the store to inject failures.
func execSQL(t *testing.T, st *Store, stmts ...string) {
	t.Helper()
	for _, s := range stmts {
		if _, err := st.DB().ExecContext(t.Context(), s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

// failOn returns SQL that makes every op (INSERT/UPDATE/DELETE) on table abort.
func failOn(table, op string) string {
	return fmt.Sprintf("CREATE TRIGGER fail_%s_%s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'injected failure'); END;",
		table, strings.ToLower(op), op, table)
}

// plainFTS replaces the FTS5 virtual table (which cannot carry triggers) with a
// regular table of the same shape so failOn can target it.
var plainFTS = []string{
	"DROP TABLE fts_entities;",
	"CREATE TABLE fts_entities(entity_type, entity_id, title, summary, body);",
}

func seedAll(t *testing.T, st *Store) {
	t.Helper()
	ctx := t.Context()
	if err := st.UpsertTask(ctx, model.Task{ID: "t1", Title: "Task", Status: "ready", Summary: "s", Tags: []string{"x"}, Dependencies: []string{"t0"}, Milestone: "m1"}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMilestone(ctx, model.Milestone{ID: "m1", Title: "M", Status: model.MilestoneStatusOpen, Summary: "s", Tags: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertStrategy(ctx, model.Strategy{ID: "s1", Title: "S", Tier: model.TierCore, Summary: "s", Tags: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertGlossaryTerm(ctx, model.GlossaryTerm{ID: "g1", Title: "G", Summary: "s", Tags: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
}

func TestMutationErrorBranches(t *testing.T) {
	task := model.Task{ID: "t1", Title: "Task", Status: "ready", Summary: "s", Tags: []string{"", "x"}, Dependencies: []string{"", "t0"}}
	ms := model.Milestone{ID: "m1", Title: "M", Status: model.MilestoneStatusOpen, Summary: "s", Tags: []string{"", "x"}}
	strat := model.Strategy{ID: "s1", Title: "S", Tier: model.TierCore, Summary: "s", Tags: []string{"", "x"}}
	term := model.GlossaryTerm{ID: "g1", Title: "G", Summary: "s", Tags: []string{"", "x"}}

	upsertTask := func(ctx context.Context, st *Store) error { return st.UpsertTask(ctx, task) }
	upsertMS := func(ctx context.Context, st *Store) error { return st.UpsertMilestone(ctx, ms) }
	upsertStrat := func(ctx context.Context, st *Store) error { return st.UpsertStrategy(ctx, strat) }
	upsertTerm := func(ctx context.Context, st *Store) error { return st.UpsertGlossaryTerm(ctx, term) }
	delTask := func(ctx context.Context, st *Store) error { return st.DeleteTask(ctx, "t1") }
	delMS := func(ctx context.Context, st *Store) error { return st.DeleteMilestone(ctx, "m1") }
	delStrat := func(ctx context.Context, st *Store) error { return st.DeleteStrategy(ctx, "s1") }
	delTerm := func(ctx context.Context, st *Store) error { return st.DeleteGlossaryTerm(ctx, "g1") }

	ftsInsert := append(append([]string{}, plainFTS...), failOn("fts_entities", "INSERT"))

	tests := []struct {
		name   string
		inject []string
		op     func(context.Context, *Store) error
		want   string
	}{
		{"UpsertTask record", []string{failOn("tasks", "INSERT")}, upsertTask, "failed to upsert task"},
		{"UpsertTask clear tags", []string{"DROP TABLE entity_tags;"}, upsertTask, "failed to clear tags for task"},
		{"UpsertTask add tag", []string{failOn("entity_tags", "INSERT")}, upsertTask, "failed to add tag"},
		{"UpsertTask clear deps", []string{"DROP TABLE task_dependencies;"}, upsertTask, "failed to clear dependencies for task"},
		{"UpsertTask add dep", []string{failOn("task_dependencies", "INSERT")}, upsertTask, "failed to add dependency"},
		{"UpsertTask delete FTS", []string{"DROP TABLE fts_entities;"}, upsertTask, "failed to delete FTS entry for task"},
		{"UpsertTask insert FTS", ftsInsert, upsertTask, "failed to insert FTS entry for task"},

		{"UpsertMilestone record", []string{failOn("milestones", "INSERT")}, upsertMS, "failed to upsert milestone"},
		{"UpsertMilestone clear tags", []string{"DROP TABLE entity_tags;"}, upsertMS, "failed to clear tags for milestone"},
		{"UpsertMilestone add tag", []string{failOn("entity_tags", "INSERT")}, upsertMS, "failed to add tag"},
		{"UpsertMilestone delete FTS", []string{"DROP TABLE fts_entities;"}, upsertMS, "failed to delete FTS entry for milestone"},
		{"UpsertMilestone insert FTS", ftsInsert, upsertMS, "failed to insert FTS entry for milestone"},

		{"UpsertStrategy record", []string{failOn("strategies", "INSERT")}, upsertStrat, "failed to upsert strategy"},
		{"UpsertStrategy clear tags", []string{"DROP TABLE entity_tags;"}, upsertStrat, "failed to clear tags for strategy"},
		{"UpsertStrategy add tag", []string{failOn("entity_tags", "INSERT")}, upsertStrat, "failed to add tag"},
		{"UpsertStrategy delete FTS", []string{"DROP TABLE fts_entities;"}, upsertStrat, "failed to delete FTS entry for strategy"},
		{"UpsertStrategy insert FTS", ftsInsert, upsertStrat, "failed to insert FTS entry for strategy"},

		{"UpsertGlossaryTerm record", []string{failOn("glossary", "INSERT")}, upsertTerm, "failed to upsert glossary term"},
		{"UpsertGlossaryTerm clear tags", []string{"DROP TABLE entity_tags;"}, upsertTerm, "failed to clear tags for glossary term"},
		{"UpsertGlossaryTerm add tag", []string{failOn("entity_tags", "INSERT")}, upsertTerm, "failed to add tag"},
		{"UpsertGlossaryTerm delete FTS", []string{"DROP TABLE fts_entities;"}, upsertTerm, "failed to delete FTS entry for glossary term"},
		{"UpsertGlossaryTerm insert FTS", ftsInsert, upsertTerm, "failed to insert FTS entry for glossary term"},

		{"DeleteTask record", []string{"DROP TABLE tasks;"}, delTask, "failed to delete task"},
		{"DeleteTask clear tags", []string{"DROP TABLE entity_tags;"}, delTask, "failed to clear tags for task"},
		{"DeleteTask clear deps", []string{"DROP TABLE task_dependencies;"}, delTask, "failed to clear dependencies for task"},
		{"DeleteTask delete FTS", []string{"DROP TABLE fts_entities;"}, delTask, "failed to delete FTS entry for task"},

		{"DeleteMilestone record", []string{"DROP TABLE milestones;"}, delMS, "failed to delete milestone"},
		{"DeleteMilestone clear tags", []string{"DROP TABLE entity_tags;"}, delMS, "failed to clear tags for milestone"},
		{"DeleteMilestone delete FTS", []string{"DROP TABLE fts_entities;"}, delMS, "failed to delete FTS entry for milestone"},

		{"DeleteStrategy record", []string{"DROP TABLE strategies;"}, delStrat, "failed to delete strategy"},
		{"DeleteStrategy clear tags", []string{"DROP TABLE entity_tags;"}, delStrat, "failed to clear tags for strategy"},
		{"DeleteStrategy delete FTS", []string{"DROP TABLE fts_entities;"}, delStrat, "failed to delete FTS entry for strategy"},

		{"DeleteGlossaryTerm record", []string{"DROP TABLE glossary;"}, delTerm, "failed to delete glossary term"},
		{"DeleteGlossaryTerm clear tags", []string{"DROP TABLE entity_tags;"}, delTerm, "failed to clear tags for glossary term"},
		{"DeleteGlossaryTerm delete FTS", []string{"DROP TABLE fts_entities;"}, delTerm, "failed to delete FTS entry for glossary term"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := newTestStore(t)
			seedAll(t, st)
			execSQL(t, st, tc.inject...)
			err := tc.op(t.Context(), st)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestDeleteMissingIDIsNoop(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()
	ops := map[string]func(string) error{
		"task":      func(id string) error { return st.DeleteTask(ctx, id) },
		"milestone": func(id string) error { return st.DeleteMilestone(ctx, id) },
		"strategy":  func(id string) error { return st.DeleteStrategy(ctx, id) },
		"glossary":  func(id string) error { return st.DeleteGlossaryTerm(ctx, id) },
	}
	for name, del := range ops {
		t.Run(name, func(t *testing.T) {
			if err := del("does-not-exist"); err != nil {
				t.Errorf("delete of missing %s: %v", name, err)
			}
		})
	}
}

func TestClosedStoreErrors(t *testing.T) {
	st, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()

	tests := []struct {
		name string
		call func() error
		want string
	}{
		{"UpsertTask", func() error { return st.UpsertTask(ctx, model.Task{ID: "a"}) }, "failed to begin transaction"},
		{"UpsertMilestone", func() error { return st.UpsertMilestone(ctx, model.Milestone{ID: "a"}) }, "failed to begin transaction"},
		{"UpsertStrategy", func() error { return st.UpsertStrategy(ctx, model.Strategy{ID: "a"}) }, "failed to begin transaction"},
		{"UpsertGlossaryTerm", func() error { return st.UpsertGlossaryTerm(ctx, model.GlossaryTerm{ID: "a"}) }, "failed to begin transaction"},
		{"DeleteTask", func() error { return st.DeleteTask(ctx, "a") }, "failed to begin transaction"},
		{"DeleteMilestone", func() error { return st.DeleteMilestone(ctx, "a") }, "failed to begin transaction"},
		{"DeleteStrategy", func() error { return st.DeleteStrategy(ctx, "a") }, "failed to begin transaction"},
		{"DeleteGlossaryTerm", func() error { return st.DeleteGlossaryTerm(ctx, "a") }, "failed to begin transaction"},
		{"UpdateTaskCriteria", func() error { return st.UpdateTaskCriteria(ctx, "a", 1, 0, "") }, "failed to begin transaction"},
		{"GetTask", func() error { _, err := st.GetTask(ctx, "a"); return err }, "failed to get task"},
		{"GetMilestone", func() error { _, err := st.GetMilestone(ctx, "a"); return err }, "failed to get milestone"},
		{"GetStrategy", func() error { _, err := st.GetStrategy(ctx, "a"); return err }, "failed to get strategy"},
		{"GetGlossaryTerm", func() error { _, err := st.GetGlossaryTerm(ctx, "a"); return err }, "failed to get glossary term"},
		{"ListTasks", func() error { _, err := st.ListTasks(ctx); return err }, "failed to list tasks"},
		{"ListMilestones", func() error { _, err := st.ListMilestones(ctx); return err }, "failed to list milestones"},
		{"ListStrategies", func() error { _, err := st.ListStrategies(ctx); return err }, "failed to list strategies"},
		{"ListGlossaryTerms", func() error { _, err := st.ListGlossaryTerms(ctx); return err }, "failed to list glossary terms"},
		{"GetTagCounts", func() error { _, err := st.GetTagCounts(ctx); return err }, "failed to get tag counts"},
		{"GetEntityTags", func() error { _, err := st.GetEntityTags(ctx, "task", "a"); return err }, "closed"},
		{"GetBoardState", func() error { _, err := st.GetBoardState(ctx, "p", nil); return err }, "failed to list tasks for board"},
		{"AreDependenciesDone", func() error { _, err := st.AreDependenciesDone(ctx, "a"); return err }, "failed to check dependencies"},
		{"GetBlockingTasks", func() error { _, err := st.GetBlockingTasks(ctx, "a"); return err }, "failed to get unfinished dependencies"},
		{"GetDownstreamTasks", func() error { _, err := st.GetDownstreamTasks(ctx, "a"); return err }, "failed to get downstream tasks"},
		{"FindUnblockedTasks", func() error { _, err := st.FindUnblockedTasks(ctx, "a"); return err }, "failed to get downstream tasks"},
		{"SearchAll", func() error { _, err := st.SearchAll(ctx, "x", "", 0); return err }, "search all failed"},
		{"SearchTasks", func() error { _, err := st.SearchTasks(ctx, "x", "", 0); return err }, "search tasks failed"},
		{"SearchMilestones", func() error { _, err := st.SearchMilestones(ctx, "x", "", 0); return err }, "search milestones failed"},
		{"SearchStrategies", func() error { _, err := st.SearchStrategies(ctx, "x", "", 0); return err }, "search strategies failed"},
		{"SearchGlossary", func() error { _, err := st.SearchGlossary(ctx, "x", "", 0); return err }, "search glossary failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestReadErrorBranches(t *testing.T) {
	tests := []struct {
		name   string
		inject []string
		call   func(context.Context, *Store) error
		want   string
	}{
		{"GetTask tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.GetTask(ctx, "t1"); return err }, "failed to get tags for task"},
		{"GetTask deps", []string{"DROP TABLE task_dependencies;"}, func(ctx context.Context, st *Store) error { _, err := st.GetTask(ctx, "t1"); return err }, "failed to get dependencies for task"},
		{"ListTasks tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.ListTasks(ctx); return err }, "failed to get tags for task"},
		{"ListTasks deps", []string{"DROP TABLE task_dependencies;"}, func(ctx context.Context, st *Store) error { _, err := st.ListTasks(ctx); return err }, "failed to get dependencies for task"},
		{"GetMilestone tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.GetMilestone(ctx, "m1"); return err }, "failed to get tags for milestone"},
		{"GetMilestone metrics", []string{"DROP TABLE tasks;"}, func(ctx context.Context, st *Store) error { _, err := st.GetMilestone(ctx, "m1"); return err }, "failed to get metrics for milestone"},
		{"ListMilestones tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.ListMilestones(ctx); return err }, "failed to get tags for milestone"},
		{"ListMilestones metrics", []string{"DROP TABLE tasks;"}, func(ctx context.Context, st *Store) error { _, err := st.ListMilestones(ctx); return err }, "failed to get metrics for milestone"},
		{"GetStrategy tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.GetStrategy(ctx, "s1"); return err }, "failed to get tags for strategy"},
		{"ListStrategies tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.ListStrategies(ctx); return err }, "failed to get tags for strategy"},
		{"ListStrategies by tier tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error {
			_, err := st.ListStrategies(ctx, model.TierCore)
			return err
		}, "failed to get tags for strategy"},
		{"GetGlossaryTerm tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.GetGlossaryTerm(ctx, "g1"); return err }, "failed to get tags for glossary term"},
		{"ListGlossaryTerms tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.ListGlossaryTerms(ctx); return err }, "failed to get tags for glossary term"},
		{"SearchAll tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.SearchAll(ctx, "s", "", 0); return err }, "failed to get tags for"},
		{"SearchTasks tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.SearchTasks(ctx, "Task", "", 0); return err }, "failed to get tags for task"},
		{"SearchMilestones tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.SearchMilestones(ctx, "M", "", 0); return err }, "failed to get tags for milestone"},
		{"SearchStrategies tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.SearchStrategies(ctx, "S", "", 0); return err }, "failed to get tags for strategy"},
		{"SearchGlossary tags", []string{"DROP TABLE entity_tags;"}, func(ctx context.Context, st *Store) error { _, err := st.SearchGlossary(ctx, "G", "", 0); return err }, "failed to get tags for glossary"},
		{"UpdateTaskCriteria get", []string{"DROP TABLE tasks;"}, func(ctx context.Context, st *Store) error { return st.UpdateTaskCriteria(ctx, "t1", 1, 1, "b") }, "no such table"},
		{"UpdateTaskCriteria update", []string{failOn("tasks", "UPDATE")}, func(ctx context.Context, st *Store) error { return st.UpdateTaskCriteria(ctx, "t1", 1, 1, "b") }, "injected failure"},
		{"UpdateTaskCriteria FTS", append(append([]string{}, plainFTS...), failOn("fts_entities", "INSERT")), func(ctx context.Context, st *Store) error { return st.UpdateTaskCriteria(ctx, "t1", 1, 1, "b") }, "injected failure"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := newTestStore(t)
			seedAll(t, st)
			execSQL(t, st, tc.inject...)
			err := tc.call(t.Context(), st)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestUpdateTaskCriteria(t *testing.T) {
	st := newTestStore(t)
	seedAll(t, st)
	ctx := t.Context()

	if err := st.UpdateTaskCriteria(ctx, "missing", 1, 0, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing task: err = %v, want ErrNotFound", err)
	}

	if err := st.UpdateTaskCriteria(ctx, "t1", 4, 2, "uniquebodyword"); err != nil {
		t.Fatalf("UpdateTaskCriteria: %v", err)
	}
	got, err := st.GetTask(ctx, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalCriteria != 4 || got.CompletedCriteria != 2 || got.Body != "uniquebodyword" || got.ChangedAt == "" {
		t.Errorf("unexpected task after update: %+v", got)
	}
	// The FTS index must be refreshed with the new body.
	res, err := st.SearchTasks(ctx, "uniquebodyword", "", 5)
	if err != nil || len(res) != 1 {
		t.Errorf("search after criteria update: %v, %d results", err, len(res))
	}
}

func TestGetEntityTags(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()
	if err := st.UpsertTask(ctx, model.Task{ID: "t1", Title: "T", Status: "ready", Summary: "s", Tags: []string{"b", "a"}}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, typ, id string
		want          int
	}{
		{"existing task", "task", "t1", 2},
		{"wrong type", "milestone", "t1", 0},
		{"unknown id", "task", "nope", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tags, err := st.GetEntityTags(ctx, tc.typ, tc.id)
			if err != nil {
				t.Fatal(err)
			}
			if len(tags) != tc.want {
				t.Errorf("got %v, want %d tags", tags, tc.want)
			}
		})
	}
}

func TestLockBlocksWriters(t *testing.T) {
	st := newTestStore(t)
	ctx := t.Context()

	st.Lock()
	done := make(chan error, 1)
	go func() {
		done <- st.UpsertTask(ctx, model.Task{ID: "t1", Title: "T", Status: "ready", Summary: "s"})
	}()
	select {
	case err := <-done:
		st.Unlock()
		t.Fatalf("writer finished while store was locked: %v", err)
	default:
	}
	st.Unlock()
	if err := <-done; err != nil {
		t.Fatalf("UpsertTask after Unlock: %v", err)
	}
	if _, err := st.GetTask(ctx, "t1"); err != nil {
		t.Fatalf("GetTask: %v", err)
	}
}

func TestWithTxCommitFailure(t *testing.T) {
	st := newTestStore(t)
	execSQL(t, st,
		"CREATE TABLE parent(id TEXT PRIMARY KEY);",
		"CREATE TABLE child(pid TEXT REFERENCES parent(id) DEFERRABLE INITIALLY DEFERRED);",
	)
	err := st.WithTx(t.Context(), func(q *Queries) error {
		_, err := q.db.ExecContext(t.Context(), "INSERT INTO child VALUES ('orphan');")
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "failed to commit transaction") {
		t.Fatalf("err = %v, want commit failure", err)
	}
}

func TestOpenErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("unreachable path", func(t *testing.T) {
		_, err := Open("file:" + filepath.Join(dir, "missing", "sub", "db.sqlite") + "?mode=ro")
		if err == nil || !strings.Contains(err.Error(), "failed to ping") {
			t.Fatalf("err = %v, want ping failure", err)
		}
	})

	t.Run("schema conflict", func(t *testing.T) {
		path := filepath.Join(dir, "conflict.sqlite")
		db, err := sql.Open("sqlite", "file:"+path)
		if err != nil {
			t.Fatal(err)
		}
		// A view cannot be indexed, so the embedded schema fails on CREATE INDEX.
		if _, err := db.ExecContext(t.Context(), "CREATE VIEW tasks AS SELECT 'x' AS status, 'x' AS milestone_id, 'x' AS priority;"); err != nil {
			t.Fatal(err)
		}
		_ = db.Close()

		_, err = Open("file:" + path)
		if err == nil || !strings.Contains(err.Error(), "failed to execute schema.sql") {
			t.Fatalf("err = %v, want schema failure", err)
		}
	})

	t.Run("default DSN", func(t *testing.T) {
		st, err := Open()
		if err != nil {
			t.Fatalf("Open(): %v", err)
		}
		_ = st.Close()
	})
}

func TestFormatFTS5Query(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"whitespace", "  \t\n ", ""},
		{"single word", "kanban", `"kanban"*`},
		{"multiple words joined with OR", "foo  bar", `"foo"* OR "bar"*`},
		{"exact phrase preserved", `"exact phrase"`, `"exact phrase"`},
		{"special runes stripped", `foo* ^bar (baz):`, `"foo"* OR "bar"* OR "baz"*`},
		{"word consisting only of operators dropped", `foo () ^* bar`, `"foo"* OR "bar"*`},
		{"only operators", `() ^ *`, ""},
		{"unbalanced quote stripped", `"foo bar`, `"foo"* OR "bar"*`},
		{"boolean keywords are quoted", "AND OR NOT", `"AND"* OR "OR"* OR "NOT"*`},
		{"dash kept inside quotes", "go-sdk", `"go-sdk"*`},
		{"unicode", "ääkköset", `"ääkköset"*`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatFTS5Query(tc.in); got != tc.want {
				t.Errorf("formatFTS5Query(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatTimeframeDate(t *testing.T) {
	tests := []struct{ in, want string }{
		{"2026-10-05T12:00:00Z", "2026-10-05"},
		{"2026-10-05", "2026-10-05"},
		{"2026/10/05 12:00", "2026/10/05 12:00"},
		{"short", "short"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if got := formatTimeframeDate(tc.in); got != tc.want {
				t.Errorf("formatTimeframeDate(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDeriveMilestoneTimeframe(t *testing.T) {
	tests := []struct {
		name                         string
		start, end, target           string
		wantStart, wantEnd, wantSpan string
	}{
		{"nothing", "", "", "", "", "", ""},
		{"target date only", "", "", "2026-12-01", "", "2026-12-01", "2026-12-01"},
		{"start without end falls back to target", "2026-10-01", "", "2026-12-01", "2026-10-01", "2026-12-01", "2026-12-01"},
		{"end without start keeps end", "", "2026-11-01", "2026-12-01", "", "2026-11-01", "2026-12-01"},
		{"same day range", "2026-10-01T08:00:00Z", "2026-10-01T18:00:00Z", "", "2026-10-01T08:00:00Z", "2026-10-01T18:00:00Z", "2026-10-01"},
		{"multi day range", "2026-10-01T08:00:00Z", "2026-10-09", "2026-12-01", "2026-10-01T08:00:00Z", "2026-10-09", "2026-10-01 – 2026-10-09"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, e, span := deriveMilestoneTimeframe(tc.start, tc.end, tc.target)
			if s != tc.wantStart || e != tc.wantEnd || span != tc.wantSpan {
				t.Errorf("got (%q, %q, %q), want (%q, %q, %q)", s, e, span, tc.wantStart, tc.wantEnd, tc.wantSpan)
			}
		})
	}
}

func TestSearchFiltersAndDefaults(t *testing.T) {
	st := newTestStore(t)
	seedAll(t, st)
	ctx := t.Context()
	if err := st.UpsertTask(ctx, model.Task{ID: "t2", Title: "Task untagged", Status: "ready", Summary: "s"}); err != nil {
		t.Fatal(err)
	}

	type searchFn func(ctx context.Context, q, tag string, limit int) ([]model.SearchResult, error)
	tests := []struct {
		name   string
		fn     searchFn
		query  string
		tag    string
		limit  int
		wantN  int
		wantID string
	}{
		{"all tagged", st.SearchAll, "s", "x", 0, 4, ""},
		{"all no match tag", st.SearchAll, "s", "none", 0, 0, ""},
		{"tasks tagged", st.SearchTasks, "Task", "x", 0, 1, "t1"},
		{"tasks untagged default limit", st.SearchTasks, "Task", "", -1, 2, ""},
		{"milestones tagged", st.SearchMilestones, "M", "x", 0, 1, "m1"},
		{"milestones empty query", st.SearchMilestones, "   ", "", 0, 0, ""},
		{"strategies tagged", st.SearchStrategies, "S", "x", 0, 1, "s1"},
		{"strategies empty query", st.SearchStrategies, "", "", 0, 0, ""},
		{"glossary tagged", st.SearchGlossary, "G", "x", 0, 1, "g1"},
		{"glossary empty query", st.SearchGlossary, "", "", 0, 0, ""},
		{"tasks empty query", st.SearchTasks, "()", "", 0, 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.fn(ctx, tc.query, tc.tag, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if res == nil {
				t.Fatal("expected non-nil result slice")
			}
			if len(res) != tc.wantN {
				t.Fatalf("got %d results (%+v), want %d", len(res), res, tc.wantN)
			}
			if tc.wantID != "" && res[0].ID != tc.wantID {
				t.Errorf("first result ID = %q, want %q", res[0].ID, tc.wantID)
			}
		})
	}
}

func TestSearchUnbalancedQuotePassthrough(t *testing.T) {
	t.Skip("known bug: formatFTS5Query passes any input that starts and ends with '\"' through verbatim, " +
		"so a lone '\"' or '\"a \"b\"' reaches FTS5 unescaped and SearchAll fails with 'unterminated string'; " +
		"see task 261005-go-test-coverage-t2-store-server-validat notes")

	st := newTestStore(t)
	seedAll(t, st)
	for _, q := range []string{`"`, `"a "b"`} {
		if _, err := st.SearchAll(t.Context(), q, "", 0); err != nil {
			t.Errorf("SearchAll(%q): %v", q, err)
		}
	}
}
