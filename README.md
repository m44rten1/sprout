# 🌱 Sprout

**Git worktrees for humans who hate clutter.**

`sprout` is a tiny CLI tool that manages your Git worktrees so you don't have to. It keeps your project directories clean by tucking worktrees away in `~/.local/share/sprout` (or `~/.sprout`), and gives you a nice fuzzy-finder interface to jump between them.

Stop `cd ../../other-repo`ing like a caveman.

## 🚀 Installation

### Homebrew

```bash
brew tap m44rten1/sprout
brew install sprout
```

### Go

```bash
go install github.com/m44rten1/sprout@latest
```

## 🛠 Usage

### Shell Completion

Enable tab completion with one command:

```bash
sprout install-completion
```

This automatically detects your shell (zsh/bash/fish) and configures completion. Restart your terminal and you're done!

Once configured, you can tab-complete:

- `sprout add <TAB>` - Shows all available branches
- `sprout open <TAB>` - Shows branches with existing worktrees
- `sprout remove <TAB>` - Shows branches with existing worktrees

📖 **[Full completion setup guide →](COMPLETION.md)**

### Add a worktree

Want to work on a new feature? Just sprout it.

```bash
sprout add feat/amazing-stuff
```

This creates a fresh worktree for `feat/amazing-stuff` in your sprout directory and sets it up for you. No more messing with `git worktree add ../../my-messy-folder/branch-name`.

If you have a `.sprout.yml` file with `on_create` hooks, they'll run automatically after creating the worktree. Your editor opens immediately so you can start browsing code while hooks run in the terminal.

**Skip hooks:**

```bash
sprout add feat/amazing-stuff --no-hooks
```

Create the worktree without running hooks, even if `.sprout.yml` exists.

```bash
sprout add feat/amazing-stuff --no-open
```

Create the worktree without opening the editor (useful for automation).

### Open a worktree

Jump back into the zone.

```bash
sprout open
```

This pops up a fuzzy finder list of your active worktrees. Pick one, and boom, you're in your editor.

If you have a `.sprout.yml` file with `on_open` hooks, they'll run automatically after opening. This keeps your worktree fresh with type-checks, codegen, etc.

**Skip hooks when opening:**

```bash
sprout open --no-hooks
```

Open the worktree without running hooks, even if `.sprout.yml` exists.

### Remove a worktree

Done with that PR? Nuke it.

```bash
sprout remove
```

Select the worktree you want to delete, and it's gone. Safe and sound.

### List worktrees

See what you've got growing.

**List worktrees for current repository:**

```bash
sprout list
```

**List worktrees from all repositories:**

```bash
sprout list --all
```

The `--all` flag shows worktrees from all your sprout-managed repositories, grouped by project. Perfect for getting a bird's-eye view of all your active work.

Output includes:

- 📦 Repository names (bold) with full paths (dim)
- Branch names (green) with worktree paths (dim)
- **Git status indicators** showing the state of each worktree:
  - 🔴 Dirty (uncommitted changes)
  - ⬆️ Ahead (unpushed commits)
  - ⬇️ Behind (needs pull)
  - 🔀 Unmerged (commits not in main/master)
- Globally aligned columns for easy scanning

Clean worktrees show no indicators. Multiple indicators can appear together (e.g., 🔴🔀).

### Repair worktrees

Fix git metadata for moved worktrees.

```bash
sprout repair
```

If you've moved your sprout directory (e.g., from `~/.sprout` to `~/.local/share/sprout`), git's internal metadata may become stale. This command runs `git worktree repair` on all discovered repositories to fix the references.

**Also prune stale references:**

```bash
sprout repair --prune
```

⚠️ **Note:** Always run `sprout repair` first without `--prune` to fix moved worktrees. Only use `--prune` after verifying the repair worked, as pruning will remove metadata for worktrees git can't find.

Note: `sprout remove` automatically prunes stale worktree references after removal.

## 🪝 Project Hooks

Sprout supports project-specific hooks that automate setup and sync tasks. Perfect for ensuring your worktrees are always ready to work with.

📖 **[Read the full hooks documentation →](HOOKS.md)**

### Configuration

Create a `.sprout.yml` file in your repository root:

```yaml
hooks:
  on_create:
    - npm ci
    - npm run build
  on_open:
    - npm run lint:types
    - npm run generate
```

**Hook types:**

- **on_create**: Runs automatically when creating a new worktree (via `sprout add`)
- **on_open**: Runs automatically when opening a worktree (via `sprout open`)

### Security

Hooks can execute arbitrary commands, so **you must explicitly trust each repository** before hooks will run.

**Trust a repository:**

```bash
sprout trust
```

**View hook status:**

```bash
sprout hooks
```

### Example Workflows

**Create new worktree with automatic bootstrap:**

```bash
sprout add feat/new-feature
# Editor opens immediately
# Hooks run automatically in terminal: npm ci, npm run build
# You can browse code while dependencies install
```

**Create worktree without running hooks:**

```bash
sprout add feat/quick-fix --no-hooks
# Worktree created, no hooks run
```

**Open worktree:**

```bash
sprout open feat/bug-fix
# Editor opens, on_open hooks run automatically in terminal
# Type-check, codegen, etc. run while you browse code
```

**Open worktree without running hooks:**

```bash
sprout open feat/bug-fix --no-hooks
# Editor opens, no hooks run
```

## 🧠 Philosophy

Your main repo folder should be for your main repo. Not a graveyard of 50 abandoned feature branches.
`sprout` enforces a clean separation:

- **Repo**: Just the bare essentials (or your main branch).
- **Work**: Happens in a dedicated sprout directory (`~/.local/share/sprout`).

It's like `virtualenv` but for your entire codebase.

Sprout follows the XDG Base Directory specification, storing worktrees in `~/.local/share/sprout` by default (or `$XDG_DATA_HOME/sprout` if that environment variable is set).

## 🤝 Contributing

Found a bug? Want to add more fertilizer? Open an issue or a PR!
