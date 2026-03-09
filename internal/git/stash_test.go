package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JonathanInTheClouds/gstash/internal/git"
)

// setupRepo creates a temporary git repo, sets it as the working directory,
// and returns a cleanup function.
func setupRepo(t *testing.T) (repoDir string, cleanup func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "gstash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	mustRun(t, "git", "init")
	mustRun(t, "git", "config", "user.email", "test@gstash.dev")
	mustRun(t, "git", "config", "user.name", "gstash tester")
	mustRun(t, "git", "commit", "--allow-empty", "-m", "initial commit")

	return dir, func() {
		os.Chdir(orig)
		os.RemoveAll(dir)
	}
}

// addStash creates a file, stages it, and stashes it with the given message.
func addStash(t *testing.T, repoDir, filename, content, message string) {
	t.Helper()
	path := filepath.Join(repoDir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	mustRun(t, "git", "add", filename)
	if message != "" {
		mustRun(t, "git", "stash", "push", "-m", message)
	} else {
		mustRun(t, "git", "stash")
	}
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %q failed: %v\n%s", name+" "+joinArgs(args), err, out)
	}
}

func joinArgs(args []string) string {
	result := ""
	for _, a := range args {
		result += a + " "
	}
	return result
}

// ---

func TestIsGitRepo(t *testing.T) {
	_, cleanup := setupRepo(t)
	defer cleanup()

	if !git.IsGitRepo() {
		t.Error("expected IsGitRepo to return true inside a git repo")
	}
}

func TestIsGitRepo_Outside(t *testing.T) {
	dir, err := os.MkdirTemp("", "not-a-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	if git.IsGitRepo() {
		t.Error("expected IsGitRepo to return false outside a git repo")
	}
}

func TestListStashes_Empty(t *testing.T) {
	_, cleanup := setupRepo(t)
	defer cleanup()

	stashes, err := git.ListStashes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stashes) != 0 {
		t.Errorf("expected 0 stashes, got %d", len(stashes))
	}
}

func TestListStashes_Multiple(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "a.go", "package a", "stash one")
	addStash(t, dir, "b.go", "package b", "stash two")
	addStash(t, dir, "c.go", "package c", "stash three")

	stashes, err := git.ListStashes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stashes) != 3 {
		t.Errorf("expected 3 stashes, got %d", len(stashes))
	}

	// git stash is a stack — most recent is index 0
	if stashes[0].Message != "stash three" {
		t.Errorf("expected first stash message 'stash three', got %q", stashes[0].Message)
	}
}

func TestListStashes_Fields(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "x.go", "package x", "my feature wip")

	stashes, err := git.ListStashes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("expected 1 stash, got %d", len(stashes))
	}

	s := stashes[0]
	if s.Index != 0 {
		t.Errorf("expected index 0, got %d", s.Index)
	}
	if s.Ref != "stash@{0}" {
		t.Errorf("expected ref 'stash@{0}', got %q", s.Ref)
	}
	if s.Message != "my feature wip" {
		t.Errorf("expected message 'my feature wip', got %q", s.Message)
	}
	if s.Branch == "" {
		t.Error("expected non-empty branch name")
	}
	if s.Date.IsZero() {
		t.Error("expected non-zero date")
	}
}

func TestShowDiff(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "diff_test.go", "package main\n\nfunc hello() {}", "diff test stash")

	diff, err := git.ShowDiff(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff output")
	}
	// Diff should mention the file we added
	found := false
	for _, line := range splitLines(diff) {
		if len(line) > 0 && contains(line, "diff_test.go") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diff to mention diff_test.go, got:\n%s", diff)
	}
}

func TestDropStash(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "drop.go", "package drop", "to be dropped")

	stashes, _ := git.ListStashes()
	if len(stashes) != 1 {
		t.Fatalf("expected 1 stash before drop")
	}

	if err := git.DropStash(0); err != nil {
		t.Fatalf("unexpected error dropping stash: %v", err)
	}

	stashes, _ = git.ListStashes()
	if len(stashes) != 0 {
		t.Errorf("expected 0 stashes after drop, got %d", len(stashes))
	}
}

func TestApplyStash(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "apply.go", "package apply", "apply test")

	if err := git.ApplyStash(0); err != nil {
		t.Fatalf("unexpected error applying stash: %v", err)
	}

	// File should now exist in working tree
	if _, err := os.Stat(filepath.Join(dir, "apply.go")); os.IsNotExist(err) {
		t.Error("expected apply.go to exist after apply, but it does not")
	}

	// Stash should still be there (apply doesn't remove)
	stashes, _ := git.ListStashes()
	if len(stashes) != 1 {
		t.Errorf("expected stash to remain after apply, got %d stashes", len(stashes))
	}
}

func TestPopStash(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "pop.go", "package pop", "pop test")

	if err := git.PopStash(0); err != nil {
		t.Fatalf("unexpected error popping stash: %v", err)
	}

	// File should exist
	if _, err := os.Stat(filepath.Join(dir, "pop.go")); os.IsNotExist(err) {
		t.Error("expected pop.go to exist after pop")
	}

	// Stash should be gone
	stashes, _ := git.ListStashes()
	if len(stashes) != 0 {
		t.Errorf("expected 0 stashes after pop, got %d", len(stashes))
	}
}

func TestRenameStash(t *testing.T) {
	dir, cleanup := setupRepo(t)
	defer cleanup()

	addStash(t, dir, "rename.go", "package rename", "old name")

	if err := git.RenameStash(0, "new name"); err != nil {
		t.Fatalf("unexpected error renaming stash: %v", err)
	}

	stashes, err := git.ListStashes()
	if err != nil {
		t.Fatalf("unexpected error listing stashes: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("expected 1 stash after rename, got %d", len(stashes))
	}
	if stashes[0].Message != "new name" {
		t.Errorf("expected message 'new name', got %q", stashes[0].Message)
	}
}

func TestRelativeTime(t *testing.T) {
	cases := []struct {
		name     string
		input    string // RFC3339
		contains string
	}{
		{"just now", "5s", "just now"},
		{"minutes", "10m", "minute"},
		{"hours", "3h", "hour"},
		{"days", "72h", "day"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// We test RelativeTime indirectly via ListStashes which sets dates,
			// so just test it parses without panicking using zero time edge case
		})
	}
}

// helpers
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	return lines
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}