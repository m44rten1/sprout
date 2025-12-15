# 🌱 Sprout

**Git worktrees for humans who hate clutter.**

`sprout` is a tiny CLI tool that manages your Git worktrees so you don't have to. It keeps your project directories clean by tucking worktrees away in `~/.sprout`, and gives you a nice fuzzy-finder interface to jump between them.

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
This creates a fresh worktree for `feat/amazing-stuff` in `~/.sprout/...` and sets it up for you. No more messing with `git worktree add ../../my-messy-folder/branch-name`.

If you have a `.sprout.yml` file with `on_create` hooks, they'll run automatically after creating the worktree. Your editor opens immediately so you can start browsing code while hooks run in the terminal.

**Skip hooks:**
```bash
sprout add feat/amazing-stuff --skip-hooks
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

```bash
sprout list
```

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
sprout add feat/quick-fix --skip-hooks
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
- **Work**: Happens in `~/.sprout`.

It's like `virtualenv` but for your entire codebase.

## 🤝 Contributing

Found a bug? Want to add more fertilizer? Open an issue or a PR!
