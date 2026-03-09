package ui

import (
	"errors"
	"strings"

	"github.com/JonathanInTheClouds/gstash/internal/git"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		previewWidth := m.width - listWidth - 3
		m.preview = viewport.New(previewWidth, m.height-4)
		m.preview.SetContent(m.diff)
		return m, nil

	case stashesLoadedMsg:
		m.stashes = msg.stashes
		m.err = nil
		// Reset cursor if it's now out of bounds
		if m.cursor >= len(m.stashes) && m.cursor > 0 {
			m.cursor = len(m.stashes) - 1
		}
		if len(m.stashes) > 0 {
			return m, loadDiff(m.stashes[m.cursor].Index)
		}
		return m, nil

	case diffLoadedMsg:
		m.diff = msg.diff
		m.preview.SetContent(m.diff)
		return m, nil

	case conflictMsg:
		// A stash apply/pop caused merge conflicts in the working tree.
		detail := ""
		if len(msg.files) > 0 {
			detail = ": " + strings.Join(msg.files, ", ")
		}
		m.warning = "⚠ Merge conflict" + detail + " — resolve manually, then `git add` the file(s)"
		m.err = nil
		// Reload in case pop removed the stash before the conflict
		stashes, err := git.ListStashes()
		if err == nil {
			m.stashes = stashes
		}
		return m, nil

	case dirtyIndexMsg:
		// The index already had unmerged files before we even tried.
		detail := ""
		if len(msg.files) > 0 {
			detail = ": " + strings.Join(msg.files, ", ")
		}
		m.warning = "⚠ Unresolved conflicts in index" + detail + " — run: git checkout HEAD -- <file>"
		m.err = nil
		return m, nil

	case errMsg:
		m.err = msg.err
		m.warning = ""
		return m, nil

	case tea.KeyMsg:
		// Any keypress clears a stale warning
		if m.warning != "" && msg.String() != "" {
			m.warning = ""
		}
		switch m.mode {
		case modeNormal:
			return m.handleNormalKeys(msg)
		case modeRename:
			return m.handleRenameKeys(msg)
		case modeConfirmDrop:
			return m.handleConfirmDropKeys(msg)
		case modeSearch:
			return m.handleSearchKeys(msg)
		}
	}

	return m, nil
}

func (m Model) handleNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.mode = modeSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m.filtered = m.stashes
		m.cursor = 0
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			active := m.activeStashes()
			return m, loadDiff(active[m.cursor].Index)
		}

	case "down", "j":
		active := m.activeStashes()
		if m.cursor < len(active)-1 {
			m.cursor++
			return m, loadDiff(active[m.cursor].Index)
		}

	case "a":
		active := m.activeStashes()
		if len(active) > 0 {
			return m, doApplyStash(active[m.cursor].Index, false)
		}

	case "p":
		active := m.activeStashes()
		if len(active) > 0 {
			return m, doApplyStash(active[m.cursor].Index, true)
		}

	case "d":
		active := m.activeStashes()
		if len(active) > 0 {
			m.mode = modeConfirmDrop
			m.message = "Drop this stash? (y/n)"
		}

	case "r":
		active := m.activeStashes()
		if len(active) > 0 {
			m.mode = modeRename
			m.renameInput.SetValue("")
			m.renameInput.Focus()
			m.message = "Rename stash (enter to confirm, esc to cancel)"
		}

	case "pgup":
		m.preview.HalfViewUp()

	case "pgdown":
		m.preview.HalfViewDown()
	}

	return m, nil
}

func (m Model) handleRenameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := m.renameInput.Value()
		m.mode = modeNormal
		m.message = ""
		if newName == "" {
			return m, nil
		}
		idx := m.activeStashes()[m.cursor].Index
		return m, doRenameStash(idx, newName)

	case "esc":
		m.mode = modeNormal
		m.message = ""
		m.renameInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmDropKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		m.message = ""
		active := m.activeStashes()
		idx := active[m.cursor].Index
		if m.cursor >= len(active)-1 && m.cursor > 0 {
			m.cursor--
		}
		return m, doDropStash(idx)

	case "n", "N", "esc":
		m.mode = modeNormal
		m.message = ""
	}

	return m, nil
}

func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.mode = modeNormal
		m.message = ""
		m.searchInput.Blur()
		if len(m.filtered) > 0 {
			return m, loadDiff(m.filtered[m.cursor].Index)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	query := strings.ToLower(m.searchInput.Value())
	m.filtered = nil
	for _, s := range m.stashes {
		if strings.Contains(strings.ToLower(s.Message), query) ||
			strings.Contains(strings.ToLower(s.Branch), query) {
			m.filtered = append(m.filtered, s)
		}
	}
	m.cursor = 0

	if len(m.filtered) > 0 {
		return m, tea.Batch(cmd, loadDiff(m.filtered[0].Index))
	}
	return m, cmd
}

// activeStashes returns filtered stashes during search, otherwise all stashes.
func (m Model) activeStashes() []git.Stash {
	if m.mode == modeSearch || (len(m.filtered) > 0 && m.searchInput.Value() != "") {
		return m.filtered
	}
	return m.stashes
}

// --- commands ---

func loadStashes() tea.Msg {
	stashes, err := git.ListStashes()
	if err != nil {
		return errMsg{err}
	}
	return stashesLoadedMsg{stashes}
}

func loadDiff(index int) tea.Cmd {
	return func() tea.Msg {
		diff, err := git.ShowDiff(index)
		if err != nil {
			return diffLoadedMsg{"(no diff available)"}
		}
		return diffLoadedMsg{diff}
	}
}

func doApplyStash(index int, pop bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if pop {
			err = git.PopStash(index)
		} else {
			err = git.ApplyStash(index)
		}
		if err != nil {
			// Dirty index — user has leftover conflicts from before
			var dirtyErr *git.DirtyIndexError
			if errors.As(err, &dirtyErr) {
				return dirtyIndexMsg{files: dirtyErr.Files}
			}
			// Fresh conflict from this apply/pop
			var conflictErr *git.ConflictError
			if errors.As(err, &conflictErr) {
				return conflictMsg{files: conflictErr.Files}
			}
			return errMsg{err}
		}
		stashes, err := git.ListStashes()
		if err != nil {
			return errMsg{err}
		}
		return stashesLoadedMsg{stashes}
	}
}

func doDropStash(index int) tea.Cmd {
	return func() tea.Msg {
		if err := git.DropStash(index); err != nil {
			return errMsg{err}
		}
		stashes, err := git.ListStashes()
		if err != nil {
			return errMsg{err}
		}
		return stashesLoadedMsg{stashes}
	}
}

func doRenameStash(index int, newName string) tea.Cmd {
	return func() tea.Msg {
		if err := git.RenameStash(index, newName); err != nil {
			return errMsg{err}
		}
		stashes, err := git.ListStashes()
		if err != nil {
			return errMsg{err}
		}
		return stashesLoadedMsg{stashes}
	}
}

// Satisfy compiler — these are used via model fields not direct calls
var _ = textinput.New
var _ = viewport.New