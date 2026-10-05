package watcher

import (
	"path/filepath"
	"testing"
)

func TestIsInDir(t *testing.T) {
	root := t.TempDir()
	tasks := filepath.Join(root, "tasks")
	p := &Pipeline{}

	tests := []struct {
		name string
		dir  string
		path string
		want bool
	}{
		{"direct child", tasks, filepath.Join(tasks, "a.md"), true},
		{"nested child", tasks, filepath.Join(tasks, "sub", "a.md"), true},
		{"prefix collision tasks2", tasks, filepath.Join(root, "tasks2", "a.md"), false},
		{"prefix collision tasks-old", tasks, filepath.Join(root, "tasks-old", "a.md"), false},
		{"parent dir", tasks, filepath.Join(root, "a.md"), false},
		{"sibling via dotdot", tasks, filepath.Join(tasks, "..", "milestones", "a.md"), false},
		{"empty dir never matches", "", filepath.Join(tasks, "a.md"), false},
		{"dot dir never matches", ".", "a.md", false},
		{"relative path against absolute dir", tasks, "a.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.isInDir(tt.dir, tt.path); got != tt.want {
				t.Fatalf("isInDir(%q, %q) = %v, want %v", tt.dir, tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/x/task.md", false},
		{"/x/config.toml", false},
		{"/x/.hidden.md", true},
		{"/x/task.md~", true},
		{"/x/task.md.tmp", true},
		{"/x/task.md.swp", true},
		{"/x/task.md.bak", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isIgnored(tt.path); got != tt.want {
				t.Fatalf("isIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
