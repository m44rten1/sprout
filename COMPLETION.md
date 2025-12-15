# Shell Completion Setup

Shell completion for `sprout` provides tab-completion when using `add`, `open`, and `remove` commands.

## Homebrew Installation

**If you installed via Homebrew, completion is automatically configured!** Just restart your terminal and tab completion will work.

Homebrew installs completion files to:
- Zsh: `$(brew --prefix)/share/zsh/site-functions/_sprout`
- Bash: `$(brew --prefix)/etc/bash_completion.d/sprout`
- Fish: `$(brew --prefix)/share/fish/vendor_completions.d/sprout.fish`

## Manual Setup Instructions

If you installed via `go install` or another method:

### Zsh (macOS)

Add this line to your `~/.zshrc`:

```bash
source <(sprout completion zsh)
```

Or for permanent setup:

```bash
sprout completion zsh > $(brew --prefix)/share/zsh/site-functions/_sprout
```

Then restart your shell or run `source ~/.zshrc`.

### Bash

Add this line to your `~/.bashrc` or `~/.bash_profile`:

```bash
source <(sprout completion bash)
```

Then restart your shell or run `source ~/.bashrc`.

### Fish

Run:

```bash
sprout completion fish | source
```

Or for permanent setup:

```bash
sprout completion fish > ~/.config/fish/completions/sprout.fish
```

## Usage

Once configured, you can use tab completion:

- `sprout add <TAB>` - Shows all available branches (local and remote)
- `sprout open <TAB>` - Shows branches with existing sprout-managed worktrees
- `sprout remove <TAB>` - Shows branches with existing sprout-managed worktrees

The interactive fzf picker (pressing Enter with no argument) still works as before.

