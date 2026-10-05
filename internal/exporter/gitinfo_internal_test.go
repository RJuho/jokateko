package exporter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGitInfo(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"

	type file struct{ path, content string }
	tests := []struct {
		name       string
		files      []file // relative to the temp dir; nil with noGit means no .git at all
		noGit      bool
		wantBranch string
		wantCommit string
	}{
		{
			name: "branch ref resolved",
			files: []file{
				{".git/HEAD", "ref: refs/heads/main\n"},
				{".git/refs/heads/main", sha + "\n"},
			},
			wantBranch: "main",
			wantCommit: sha,
		},
		{
			name: "nested branch name",
			files: []file{
				{".git/HEAD", "ref: refs/heads/feature/x"},
				{".git/refs/heads/feature/x", sha},
			},
			wantBranch: "feature/x",
			wantCommit: sha,
		},
		{
			name:       "branch ref missing (e.g. packed refs)",
			files:      []file{{".git/HEAD", "ref: refs/heads/dev\n"}},
			wantBranch: "dev",
			wantCommit: "",
		},
		{
			name:       "detached HEAD with 40-char SHA",
			files:      []file{{".git/HEAD", sha + "\n"}},
			wantCommit: sha,
		},
		{
			name:  "HEAD with unexpected content",
			files: []file{{".git/HEAD", "garbage"}},
		},
		{
			name:  "HEAD with short SHA is ignored",
			files: []file{{".git/HEAD", sha[:12]}},
		},
		{
			name:  "no .git directory",
			noGit: true,
		},
		{
			name:  ".git is a file (worktree pointer) without HEAD",
			files: []file{{".git", "gitdir: /elsewhere\n"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				p := filepath.Join(dir, filepath.FromSlash(f.path))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(f.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			branch, commit := resolveGitInfo(dir)
			if branch != tt.wantBranch || commit != tt.wantCommit {
				t.Errorf("resolveGitInfo = (%q, %q), want (%q, %q)", branch, commit, tt.wantBranch, tt.wantCommit)
			}
		})
	}
}

func TestResolveGitInfoBranchRefIsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "refs", "heads", "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main"), 0o644); err != nil {
		t.Fatal(err)
	}
	branch, commit := resolveGitInfo(dir)
	if branch != "main" || commit != "" {
		t.Errorf("got (%q, %q)", branch, commit)
	}
}
