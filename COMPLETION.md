# Shell Completion Setup

Shell completion for `sprout` provides tab-completion when using `add`, `open`, and `remove` commands.

## Automatic Setup (Recommended)

Run this one command:

```bash
sprout install-completion
```

This will:
- Detect your shell (zsh/bash/fish)
- Find your shell config file (~/.zshrc, ~/.bashrc, etc.)
- Add the necessary completion setup
- Create a backup of your config file
- Work with both Homebrew and non-Homebrew installations

**Preview before installing:**
```bash
sprout install-completion --dry-run
```

After installation, restart your terminal or run:
```bash
source ~/.zshrc  # or ~/.bashrc, etc.
```

## Manual Setup Instructions (Advanced)

If you prefer to configure manually:

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

