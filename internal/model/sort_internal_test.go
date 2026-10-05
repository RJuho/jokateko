package model

import "testing"

func TestCompareCustomField(t *testing.T) {
	const (
		older = "2026-09-01T00:00:00Z"
		newer = "2026-09-05T00:00:00Z"
	)

	tests := []struct {
		name      string
		a, b      Task
		field     string
		direction string
		want      int
	}{
		// target_at: both set
		{"target_at both asc earlier first", Task{TargetAt: "2026-09-10"}, Task{TargetAt: "2026-09-20"}, "target_at", "asc", -1},
		{"target_at both desc later first", Task{TargetAt: "2026-09-10"}, Task{TargetAt: "2026-09-20"}, "target_at", "desc", 1},
		{"target_at both equal falls back to changed_at desc", Task{TargetAt: "2026-09-10", ChangedAt: newer}, Task{TargetAt: "2026-09-10", ChangedAt: older}, "target_at", "asc", -1},
		{"target_at both equal desc falls back to changed_at desc", Task{TargetAt: "2026-09-10", ChangedAt: older}, Task{TargetAt: "2026-09-10", ChangedAt: newer}, "target_at", "desc", 1},
		// target_at: only one set (direction-independent)
		{"target_at only a set asc", Task{TargetAt: "2026-09-10"}, Task{}, "target_at", "asc", -1},
		{"target_at only a set desc", Task{TargetAt: "2026-09-10"}, Task{}, "target_at", "desc", -1},
		{"target_at only b set asc", Task{}, Task{TargetAt: "2026-09-10"}, "target_at", "asc", 1},
		{"target_at only b set desc", Task{TargetAt: "  "}, Task{TargetAt: "2026-09-10"}, "target_at", "desc", 1},
		// target_at: none set
		{"target_at none set changed_at desc", Task{ChangedAt: newer}, Task{ChangedAt: older}, "target_at", "asc", -1},
		{"target_at none set desc changed_at desc", Task{ChangedAt: older}, Task{ChangedAt: newer}, "target_at", "desc", 1},
		{"target_at none set equal", Task{}, Task{}, "target_at", "asc", 0},
		// changed_at
		{"changed_at asc", Task{ChangedAt: older}, Task{ChangedAt: newer}, "changed_at", "asc", -1},
		{"changed_at desc", Task{ChangedAt: older}, Task{ChangedAt: newer}, "changed_at", "desc", 1},
		{"changed_at empty direction is asc", Task{ChangedAt: newer}, Task{ChangedAt: older}, "changed_at", "", 1},
		// created_at
		{"created_at asc", Task{CreatedAt: older}, Task{CreatedAt: newer}, "created_at", "asc", -1},
		{"created_at desc", Task{CreatedAt: older}, Task{CreatedAt: newer}, "created_at", "desc", 1},
		// title (case-insensitive)
		{"title asc", Task{Title: "apple"}, Task{Title: "Banana"}, "title", "asc", -1},
		{"title desc", Task{Title: "apple"}, Task{Title: "Banana"}, "title", "desc", 1},
		{"title equal ignoring case", Task{Title: "Apple"}, Task{Title: "apple"}, "title", "desc", 0},
		// id
		{"id asc", Task{ID: "a"}, Task{ID: "b"}, "id", "asc", -1},
		{"id desc", Task{ID: "a"}, Task{ID: "b"}, "id", "desc", 1},
		// priority tie-break
		{"priority falls back to changed_at desc", Task{ChangedAt: newer}, Task{ChangedAt: older}, "priority", "asc", -1},
		// unknown
		{"unknown field", Task{ID: "a", ChangedAt: older}, Task{ID: "b", ChangedAt: newer}, "bogus", "asc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareCustomField(tt.a, tt.b, tt.field, tt.direction); got != tt.want {
				t.Errorf("compareCustomField(%q, %q) = %d, want %d", tt.field, tt.direction, got, tt.want)
			}
		})
	}
}
