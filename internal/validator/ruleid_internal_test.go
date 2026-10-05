package validator

import "testing"

func TestConfigRuleID(t *testing.T) {
	tests := []struct{ msg, want string }{
		{"CFG-005: path escapes workspace root", "CFG-005"},
		{"CFG-003:", "CFG-003"},
		{"CFG-005", "CFG-001"}, // exactly 7 characters must not index out of range
		{"short", "CFG-001"},
		{"", "CFG-001"},
		{"no rule prefix here", "CFG-001"},
	}
	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			if got := configRuleID(tc.msg); got != tc.want {
				t.Errorf("configRuleID(%q) = %q, want %q", tc.msg, got, tc.want)
			}
		})
	}
}
