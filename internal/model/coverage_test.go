package model_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/RJuho/jokateko/internal/model"
)

func ids(tasks []model.Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}

func TestCompareTasksForColumn_DoneCustomSort(t *testing.T) {
	col := model.Column{ID: "done", SortBy: " Title ", SortDirection: "ASC"}
	tasks := []model.Task{
		{ID: "t1", Status: "done", Title: "Zulu"},
		{ID: "t2", Status: "done", Title: "Alpha"},
		{ID: "t3", Status: "done", Title: "alpha"}, // ties with t2 -> ID ascending
	}
	model.SortTasksForColumn(tasks, col)
	if got, want := ids(tasks), []string{"t2", "t3", "t1"}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestSortTasksByColumnOrder_UnknownColumns(t *testing.T) {
	columns := []model.Column{
		{ID: "backlog"},
		{ID: "ready", SortBy: "id", SortDirection: "desc"},
	}
	tasks := []model.Task{
		{ID: "u-zeta-1", Status: "zeta", ChangedAt: "2026-09-01T00:00:00Z"},
		{ID: "u-alpha-1", Status: "alpha"},
		{ID: "r-a", Status: "ready"},
		{ID: "u-zeta-2", Status: "zeta", ChangedAt: "2026-09-05T00:00:00Z"},
		{ID: "b-1", Status: "backlog"},
		{ID: "r-b", Status: "ready"},
	}
	orig := slices.Clone(tasks)

	got := model.SortTasksByColumnOrder(tasks, columns)

	want := []string{
		"b-1",
		"r-b", "r-a", // ready column sorts by id desc
		"u-alpha-1",            // unknown columns after known, by status name
		"u-zeta-2", "u-zeta-1", // within unknown column: default rules (changed_at desc)
	}
	if !slices.Equal(ids(got), want) {
		t.Errorf("order = %v, want %v", ids(got), want)
	}
	if !reflect.DeepEqual(tasks, orig) {
		t.Error("SortTasksByColumnOrder must not mutate its input")
	}
}

func TestSortTasksByColumnOrder_Empty(t *testing.T) {
	if got := model.SortTasksByColumnOrder(nil, nil); len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestMilestoneFrontmatterAndHasTag(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		fm := model.Milestone{Title: "M"}.Frontmatter()
		if fm.Status != model.MilestoneStatusOpen {
			t.Errorf("status = %q, want open", fm.Status)
		}
		if fm.Tags == nil || len(fm.Tags) != 0 {
			t.Errorf("tags = %#v, want empty non-nil slice", fm.Tags)
		}
	})

	t.Run("explicit values", func(t *testing.T) {
		ms := model.Milestone{
			Title:      "MVP",
			Status:     model.MilestoneStatusClosed,
			TargetDate: "2026-12-01",
			Tags:       []string{"release"},
			Summary:    "Ship it",
		}
		want := model.MilestoneFrontmatter{
			Title:      "MVP",
			Status:     model.MilestoneStatusClosed,
			TargetDate: "2026-12-01",
			Tags:       []string{"release"},
			Summary:    "Ship it",
		}
		if got := ms.Frontmatter(); !reflect.DeepEqual(got, want) {
			t.Errorf("Frontmatter() = %+v, want %+v", got, want)
		}
		if !ms.HasTag("release") {
			t.Error("expected HasTag(release)")
		}
		if ms.HasTag("other") {
			t.Error("did not expect HasTag(other)")
		}
		if (model.Milestone{}).HasTag("") {
			t.Error("milestone without tags must not have empty tag")
		}
	})
}

func TestMilestoneTargetDateFallback(t *testing.T) {
	tests := []struct {
		name          string
		ms            model.Milestone
		tasks         []model.Task
		wantTimeframe string
		wantEnd       string
	}{
		{
			name:          "target date used when no task targets",
			ms:            model.Milestone{ID: "m", TargetDate: "2026-12-01"},
			tasks:         []model.Task{{ID: "t", Milestone: "m"}},
			wantTimeframe: "2026-12-01",
			wantEnd:       "2026-12-01",
		},
		{
			name:          "no dates at all",
			ms:            model.Milestone{ID: "m", TargetTimeframe: "stale"},
			wantTimeframe: "",
		},
		{
			name:          "non-ISO task targets kept verbatim",
			ms:            model.Milestone{ID: "m"},
			tasks:         []model.Task{{ID: "a", Milestone: "m", TargetAt: "Q1"}, {ID: "b", Milestone: "m", TargetAt: "Q3"}},
			wantTimeframe: "Q1 – Q3",
			wantEnd:       "Q3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := tt.ms
			ms.RecalculateProgress(tt.tasks)
			if ms.TargetTimeframe != tt.wantTimeframe {
				t.Errorf("TargetTimeframe = %q, want %q", ms.TargetTimeframe, tt.wantTimeframe)
			}
			if ms.TargetEndAt != tt.wantEnd {
				t.Errorf("TargetEndAt = %q, want %q", ms.TargetEndAt, tt.wantEnd)
			}
		})
	}
}

// roundTrip marshals v to TOML and unmarshals it into a fresh value of the same type.
func roundTrip[T any](t *testing.T, v T) T {
	t.Helper()
	data, err := toml.Marshal(v)
	if err != nil {
		t.Fatalf("toml.Marshal: %v", err)
	}
	var out T
	if err := toml.Unmarshal(data, &out); err != nil {
		t.Fatalf("toml.Unmarshal: %v\n%s", err, data)
	}
	return out
}

func TestFrontmatterRoundTrips(t *testing.T) {
	t.Run("task defaults", func(t *testing.T) {
		fm := model.Task{Title: "T", Status: "backlog"}.Frontmatter()
		if fm.Priority != model.PriorityMedium {
			t.Errorf("priority = %q, want medium", fm.Priority)
		}
		if fm.Tags == nil || fm.Dependencies == nil {
			t.Errorf("expected non-nil tags and deps, got %#v / %#v", fm.Tags, fm.Dependencies)
		}
	})

	t.Run("task", func(t *testing.T) {
		task := model.Task{
			Title:        "Task",
			Status:       "ready",
			Priority:     model.PriorityCritical,
			Milestone:    "m1",
			Tags:         []string{"a", "b"},
			Summary:      "sum",
			Dependencies: []string{"dep-1"},
			CreatedAt:    "2026-09-01T00:00:00Z",
			ChangedAt:    "2026-09-02T00:00:00Z",
			TargetAt:     "2026-09-03",
		}
		fm := task.Frontmatter()
		if got := roundTrip(t, fm); !reflect.DeepEqual(got, fm) {
			t.Errorf("round trip = %+v, want %+v", got, fm)
		}
	})

	t.Run("milestone", func(t *testing.T) {
		fm := model.Milestone{Title: "M", Tags: []string{"x"}, Summary: "s", TargetDate: "2026-10-01"}.Frontmatter()
		if got := roundTrip(t, fm); !reflect.DeepEqual(got, fm) {
			t.Errorf("round trip = %+v, want %+v", got, fm)
		}
	})

	t.Run("strategy invalid tier defaults to core", func(t *testing.T) {
		fm := model.Strategy{Title: "S", Tier: model.Tier(0)}.Frontmatter()
		if fm.Tier != model.TierCore {
			t.Errorf("tier = %d, want %d", fm.Tier, model.TierCore)
		}
		if fm.Tags == nil {
			t.Error("expected non-nil tags")
		}
	})

	t.Run("strategy", func(t *testing.T) {
		fm := model.Strategy{Title: "S", Tier: model.TierImplementation, Tags: []string{"go"}, Summary: "s"}.Frontmatter()
		if got := roundTrip(t, fm); !reflect.DeepEqual(got, fm) {
			t.Errorf("round trip = %+v, want %+v", got, fm)
		}
	})

	t.Run("glossary nil tags", func(t *testing.T) {
		fm := model.GlossaryTerm{Title: "G"}.Frontmatter()
		if fm.Tags == nil || len(fm.Tags) != 0 {
			t.Errorf("tags = %#v, want empty non-nil slice", fm.Tags)
		}
	})

	t.Run("glossary", func(t *testing.T) {
		fm := model.GlossaryTerm{Title: "G", Tags: []string{"core"}, Summary: "s"}.Frontmatter()
		if got := roundTrip(t, fm); !reflect.DeepEqual(got, fm) {
			t.Errorf("round trip = %+v, want %+v", got, fm)
		}
	})
}
