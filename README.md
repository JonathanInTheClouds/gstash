# gstash

A terminal UI for managing git stashes — browse, preview, apply, drop, and rename stashes without memorizing `stash@{n}` syntax.

## Features

- 📋 **List & navigate** stashes with branch and relative timestamp
- 👁️ **Diff preview** pane with syntax-colored output
- ✅ **Apply / Pop / Drop** with a single keypress
- ✏️ **Rename** stashes so "WIP" actually means something
- 🔍 **Search / filter** stashes by message or branch name
- ⚠️ **Conflict detection** — surfaces merge conflicts clearly instead of silently failing

## Keybindings

| Key         | Action                        |
| ----------- | ----------------------------- |
| `↑` / `k`   | Move up                       |
| `↓` / `j`   | Move down                     |
| `/`         | Search/filter stashes         |
| `a`         | Apply stash (keep it)         |
| `p`         | Pop stash (apply and remove)  |
| `d`         | Drop stash (confirm required) |
| `r`         | Rename stash                  |
| `PgUp/PgDn` | Scroll diff preview           |
| `esc`       | Cancel current action         |
| `q`         | Quit                          |

## Installation

```bash
git clone https://github.com/user/gstash
cd gstash
go build -ldflags "-X main.Version=1.0.0" -o gstash ./cmd/gstash
mv gstash /usr/local/bin/
```

Check your version:

```bash
gstash --version
# gstash version 1.0.0
```

## Running Tests

```bash
go test ./internal/git/...
```

## Usage

Run `gstash` from inside any git repository.

```bash
cd my-project
gstash
```

## Requirements

- Go 1.21+
- git
