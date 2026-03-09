package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ConflictError is returned when a stash apply/pop produces merge conflicts.
type ConflictError struct {
	Files []string
}

func (e *ConflictError) Error() string {
	if len(e.Files) == 0 {
		return "merge conflict — resolve conflicts before continuing"
	}
	return fmt.Sprintf("merge conflict in: %s", strings.Join(e.Files, ", "))
}

// DirtyIndexError is returned when the index has unmerged files from a previous conflict.
type DirtyIndexError struct {
	Files []string
}

func (e *DirtyIndexError) Error() string {
	return fmt.Sprintf("unresolved conflicts in index: %s", strings.Join(e.Files, ", "))
}

// Stash represents a single git stash entry.
type Stash struct {
	Index   int
	Ref     string
	Branch  string
	Message string
	Date    time.Time
	RawDate string
}

// ListStashes returns all stashes in the current git repo.
func ListStashes() ([]Stash, error) {
	out, err := run("git", "stash", "list", "--format=%gd|%gs|%ci")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var stashes []Stash

	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}

		ref := strings.TrimSpace(parts[0])
		subject := strings.TrimSpace(parts[1])
		rawDate := strings.TrimSpace(parts[2])

		branch, message := parseSubject(subject)
		date, _ := time.Parse("2006-01-02 15:04:05 -0700", rawDate)

		stashes = append(stashes, Stash{
			Index:   i,
			Ref:     ref,
			Branch:  branch,
			Message: message,
			Date:    date,
			RawDate: rawDate,
		})
	}

	return stashes, nil
}

// ShowDiff returns the diff for a stash entry.
func ShowDiff(index int) (string, error) {
	return run("git", "stash", "show", "-p", "--color=never", fmt.Sprintf("stash@{%d}", index))
}

// UnmergedFiles returns any files currently in a conflicted state in the index.
func UnmergedFiles() ([]string, error) {
	out, err := run("git", "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// ApplyStash applies a stash without removing it.
func ApplyStash(index int) error {
	if err := checkDirtyIndex(); err != nil {
		return err
	}
	out, gitErr := runWithOutput("git", "stash", "apply", fmt.Sprintf("stash@{%d}", index))
	if gitErr != nil {
		return parseConflictError(out, gitErr)
	}
	return nil
}

// PopStash applies a stash and removes it.
func PopStash(index int) error {
	if err := checkDirtyIndex(); err != nil {
		return err
	}
	out, gitErr := runWithOutput("git", "stash", "pop", fmt.Sprintf("stash@{%d}", index))
	if gitErr != nil {
		return parseConflictError(out, gitErr)
	}
	return nil
}

// DropStash removes a stash entry.
func DropStash(index int) error {
	_, err := run("git", "stash", "drop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// RenameStash renames a stash by re-storing it with a new message.
func RenameStash(index int, newMessage string) error {
	ref, err := run("git", "rev-parse", fmt.Sprintf("stash@{%d}", index))
	if err != nil {
		return fmt.Errorf("could not resolve stash ref: %w", err)
	}
	ref = strings.TrimSpace(ref)

	if err := DropStash(index); err != nil {
		return fmt.Errorf("could not drop stash: %w", err)
	}

	_, err = run("git", "stash", "store", "-m", newMessage, ref)
	if err != nil {
		return fmt.Errorf("could not store stash with new message: %w", err)
	}

	return nil
}

// IsGitRepo checks whether the current directory is inside a git repo.
func IsGitRepo() bool {
	_, err := run("git", "rev-parse", "--git-dir")
	return err == nil
}

// --- helpers ---

func checkDirtyIndex() error {
	files, err := UnmergedFiles()
	if err != nil {
		return err
	}
	if len(files) > 0 {
		return &DirtyIndexError{Files: files}
	}
	return nil
}

// run executes a command and returns output. On failure returns ("", error).
func run(name string, args ...string) (string, error) {
	out, err := runWithOutput(name, args...)
	if err != nil {
		return "", err
	}
	return out, nil
}

// runWithOutput always returns the combined output even on non-zero exit.
// This is critical for conflict detection — git outputs conflict info to stdout
// even when it exits with an error code.
func runWithOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return outStr, fmt.Errorf("%s", outStr)
	}
	return string(out), nil
}

// parseConflictError checks git output for conflict markers and returns
// a typed ConflictError, or the original error if no conflict is detected.
func parseConflictError(output string, original error) error {
	lower := strings.ToLower(output)
	if !strings.Contains(lower, "conflict") && !strings.Contains(lower, "merge failed") {
		return original
	}

	var files []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		// Match: "CONFLICT (content): Merge conflict in path/to/file.go"
		if strings.HasPrefix(trimmed, "CONFLICT") && strings.Contains(trimmed, " in ") {
			parts := strings.SplitN(trimmed, " in ", 2)
			if len(parts) == 2 {
				files = append(files, strings.TrimSpace(parts[1]))
			}
		}
	}
	return &ConflictError{Files: files}
}

func parseSubject(subject string) (branch, message string) {
	for _, prefix := range []string{"WIP on ", "On "} {
		if strings.HasPrefix(subject, prefix) {
			rest := strings.TrimPrefix(subject, prefix)
			parts := strings.SplitN(rest, ": ", 2)
			if len(parts) == 2 {
				return parts[0], parts[1]
			}
			return rest, subject
		}
	}
	return "unknown", subject
}

// RelativeTime returns a human-friendly relative time string.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return strconv.Itoa(m) + " minute" + plural(m) + " ago"
	case d < 24*time.Hour:
		h := int(d.Hours())
		return strconv.Itoa(h) + " hour" + plural(h) + " ago"
	case d < 7*24*time.Hour:
		day := int(d.Hours() / 24)
		return strconv.Itoa(day) + " day" + plural(day) + " ago"
	default:
		return t.Format("Jan 2, 2006")
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}