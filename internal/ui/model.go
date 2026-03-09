package ui

import (
	"github.com/JonathanInTheClouds/gstash/internal/git"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeNormal mode = iota
	modeRename
	modeConfirmDrop
	modeSearch
)

// Model is the main Bubble Tea model.
type Model struct {
	stashes     []git.Stash
	filtered    []git.Stash
	cursor      int
	diff        string
	mode        mode
	renameInput textinput.Model
	searchInput textinput.Model
	preview     viewport.Model
	width       int
	height      int
	message     string // status bar message
	warning     string // yellow warning (conflicts etc)
	err         error
}

// Messages
type stashesLoadedMsg struct{ stashes []git.Stash }
type diffLoadedMsg struct{ diff string }
type errMsg struct{ err error }
type conflictMsg struct {
	files []string
}
type dirtyIndexMsg struct {
	files []string
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "new stash name..."
	ti.CharLimit = 100

	si := textinput.New()
	si.Placeholder = "search stashes..."
	si.CharLimit = 100

	return Model{
		renameInput: ti,
		searchInput: si,
		mode:        modeNormal,
	}
}

func (m Model) Init() tea.Cmd {
	return loadStashes
}