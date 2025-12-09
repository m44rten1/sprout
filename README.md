# 🌱 Sprout

**Git worktrees for humans who hate clutter.**

`sprout` is a tiny CLI tool that manages your Git worktrees so you don't have to. It keeps your project directories clean by tucking worktrees away in `~/.sprout`, and gives you a nice fuzzy-finder interface to jump between them.

Stop `cd ../../other-repo`ing like a caveman.

## 🚀 Installation

```bash
go install github.com/m44rten1/sprout@latest
```

## 🛠 Usage

### Add a worktree
Want to work on a new feature? Just sprout it.

```bash
sprout add feat/amazing-stuff
```
This creates a fresh worktree for `feat/amazing-stuff` in `~/.sprout/...` and sets it up for you. No more messing with `git worktree add ../../my-messy-folder/branch-name`.

### Open a worktree
Jump back into the zone.

```bash
sprout open
```
This pops up a fuzzy finder list of your active worktrees. Pick one, and boom, you're in your editor.

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

### Prune
Clean up the dead leaves.

```bash
sprout prune
```

## 🧠 Philosophy

Your main repo folder should be for your main repo. Not a graveyard of 50 abandoned feature branches.
`sprout` enforces a clean separation:
- **Repo**: Just the bare essentials (or your main branch).
- **Work**: Happens in `~/.sprout`.

It's like `virtualenv` but for your entire codebase.

## 🤝 Contributing

Found a bug? Want to add more fertilizer? Open an issue or a PR!
