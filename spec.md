# sprout – Git worktree helper

## Overview

**sprout** is a lightweight Go CLI tool for managing Git worktrees.

It provides an ergonomic interface for creating, opening, listing, and removing worktrees, with interactive selection and smart handling of remote branches.

sprout's main goals:

- Make Git worktrees trivial to use in day-to-day development.
- Keep project directories clean by storing worktrees under `~/.sprout`.
- Offer a small, predictable command surface (`sprout add`, `sprout open`, `sprout remove`, …).
- Integrate nicely with editors (Cursor, VS Code, …) without being editor-specific.
- Support automated setup via hooks for consistent worktree environments.

---

## Worktree storage layout

sprout never creates worktrees as siblings of the main repo.
Instead, all worktrees live under a central root directory.

**Sprout root location:**
- `$XDG_DATA_HOME/sprout` (if the `$XDG_DATA_HOME` environment variable is set)
- `$HOME/.local/share/sprout` (default, XDG-compliant)

sprout follows the XDG Base Directory specification, checking `$XDG_DATA_HOME` first, then falling back to the standard `~/.local/share/sprout` location.

Within the sprout root, worktrees are grouped by repository identity:

```text
$HOME/.sprout/<repo-slug>-<repo-id>/<branch-path>/<repo-slug>/
```

	•	repo-slug: basename of the repo root
	•	Example: repo at /Users/you/Projects/vl-widgets → repo-slug = vl-widgets
	•	repo-id: a short, stable identifier derived from the repo path
	•	Example: repo-id = sha1(<absolute-repo-root>)[:8]
	•	Ensures two different clones with the same name don't collide
	•	branch-path: the Git branch name, used as a path
	•	Example: branch bugfix/handover-double-message
→ directory: bugfix/handover-double-message
	•	repo-slug (again): the repo name is appended at the end to create the final worktree directory
	•	sprout must ensure intermediate directories exist (mkdir -p).

Examples

Repo root:

```
/Users/maarten/Documents/Projects/vl-widgets
```

Branch:

```
bugfix/handover-double-message
```

Actual layout:

```
$HOME/.sprout/
  vl-widgets-a1b2c3d4/
    bugfix/
      handover-double-message/
        vl-widgets/
          # worktree files here
```

sprout never touches the user's project folder structure beyond reading Git metadata.

⸻

## Repo detection

For any command, sprout must:
1. Run from somewhere inside a Git working directory.
2. Resolve the repo root via:

```bash
git rev-parse --show-toplevel
```

3. Use that absolute path to compute:
   - `repo-slug` = basename(repo-root)
   - `repo-id` = sha1(repo-root)[:8]
   - `repo-root` itself for Git commands that must run in the main worktree.

sprout should fail clearly if it's not inside a Git repo.

⸻

## Hooks System

sprout supports automated setup and sync tasks via hooks defined in `.sprout.yml` at the repository root.

### Configuration File

Create a `.sprout.yml` file in your repository root:

```yaml
hooks:
  on_create:
    - npm ci
    - npm run build
  on_open:
    - npm run lint:types
```

### Hook Types

**`on_create`**: Runs automatically after creating a new worktree via `sprout add`
- Ideal for: dependency installation, builds, database migrations, code generation

**`on_open`**: Runs automatically when opening a worktree via `sprout open`
- Ideal for: type checking, lightweight sync operations, code generation

### Security Model

Hooks execute arbitrary shell commands. For security, repositories must be explicitly trusted before hooks will run:

```bash
sprout trust
```

If hooks are defined but the repo is not trusted, sprout displays an error message and exits without running hooks.

### Hook Execution

- Commands run sequentially via `sh -lc "<command>"`
- If a command fails, subsequent commands are skipped
- Editor opens immediately, then hooks run in the terminal (allows working while hooks execute)
- Can be skipped with `--no-hooks`

### Environment Variables

When hooks run, the following variables are available:

- `SPROUT_REPO_ROOT`: Path to the git repository root
- `SPROUT_WORKTREE_PATH`: Path to the current worktree
- `SPROUT_HOOK_TYPE`: Either `on_create` or `on_open`

### Config Fallback

sprout looks for `.sprout.yml` in:
1. Current worktree path first (worktree-specific config)
2. Main worktree path as fallback (shared config, useful for gitignored configs)

### Detailed Documentation

See [HOOKS.md](HOOKS.md) for comprehensive documentation including examples, troubleshooting, and best practices.

⸻

## Commands

### 1. sprout add [branch]

Create a new worktree for `[branch]` under `~/.sprout`.

**Interactive Mode:**

If no branch is provided, sprout displays an interactive fuzzy finder with all available branches (local and remote) for selection.

```bash
sprout add
# Opens fuzzy finder to select a branch
```

**Direct Mode:**

```bash
sprout add feat/new-feature
```

**Behavior:**

1. Determine paths:

```
repo-root      = git rev-parse --show-toplevel
repo-slug      = basename(repo-root)
repo-id        = sha1(repo-root)[:8]
worktree-root  = $HOME/.sprout/<repo-slug>-<repo-id>
worktree-path  = <worktree-root>/<branch>/<repo-slug>
```

2. Check for `.sprout.yml` with `on_create` hooks:
   - If hooks exist and `--no-hooks` not set:
     - Verify repository is trusted (see `sprout trust`)
     - If not trusted, show helpful error message and exit
   - If hooks exist and `--no-hooks` is set, skip hook execution

3. Check if worktree already exists:
   - If it exists, open it in the editor

4. Create the worktree using Git with the following fallback logic:
   - If branch exists locally: checkout that branch
   - Else if `origin/<branch>` exists: create local branch from remote with `--no-track` flag
   - Else if `origin/main` exists: create new branch from `origin/main` with `--no-track`
   - Else if local `main` exists: create new branch from `main` with `--no-track`
   - Else: create new branch from `HEAD` with `--no-track`

   The `--no-track` flag prevents automatic upstream configuration for new branches.

5. If hooks should run (repo is trusted and `.sprout.yml` has `on_create` hooks):
   - Open the worktree in an editor immediately (unless `--no-open` is set)
   - Run `on_create` hooks in the terminal
   - This allows browsing code while dependencies install

6. If no hooks to run and `--no-open` not set:
   - Open the worktree in an editor after creation

**Flags:**
- `--no-hooks`: Skip running `on_create` hooks even if `.sprout.yml` exists
- `--no-open`: Skip opening the worktree in an editor

**Notes:**
- sprout creates all parent directories automatically
- If the worktree already exists, sprout opens it instead of failing
- See [HOOKS.md](HOOKS.md) for detailed hook documentation

⸻

### 2. sprout open [branch-or-path]

Open an existing worktree in an editor.

**Modes:**

1. **Interactive** (`sprout open`)
   - Lists all sprout-managed worktrees for the current repo (excludes main worktree)
   - Display format:

   ```
   <branch>                        <path>
   bugfix/handover-double-message  /Users/.../.sprout/vl-widgets-a1b2c3d4/bugfix/handover-double-message/vl-widgets
   feat/new-widget                 /Users/.../.sprout/vl-widgets-a1b2c3d4/feat/new-widget/vl-widgets
   ```

   - Use fuzzy finder to pick one
   - Open the selected path in an editor

2. **Path** (`sprout open /some/path`)
   - If the argument is an existing directory, treat it as the worktree path and open it

3. **Branch** (`sprout open <branch>`)
   - Compute `worktree-path = $HOME/.sprout/<repo-slug>-<repo-id>/<branch>/<repo-slug>`
   - Open it if it exists; otherwise, show a helpful error ("no worktree for this branch")

**Behavior:**

1. Check for `.sprout.yml` with `on_open` hooks:
   - If hooks exist and `--no-hooks` not set:
     - Verify repository is trusted (see `sprout trust`)
     - If not trusted, show helpful error message and exit
   - If hooks exist and `--no-hooks` is set, skip hook execution

2. Open the worktree in an editor immediately

3. Run `on_open` hooks automatically if:
   - `.sprout.yml` exists with `on_open` hooks
   - Repository is trusted
   - `--no-hooks` flag not set

**Note:** `on_open` hooks run after the editor is opened, allowing you to start browsing code while hooks execute in the terminal (e.g., type checking, code generation).

**Flags:**
- `--no-hooks`: Skip running `on_open` hooks even if `.sprout.yml` exists

⸻

### 3. sprout remove [branch-or-path]

Remove an existing worktree.

**Modes:**

1. **Interactive** (`sprout remove`)
   - Lists all sprout-managed worktrees for the current repo (excludes main worktree)
   - Same listing mechanism as `sprout open`
   - After selecting a worktree, removes it via:

   ```bash
   git -C <repo-root> worktree remove <path>
   ```

   - Main repo root is never shown in the list

2. **Path** (`sprout remove /some/path`)
   - If argument is an existing directory, treat it as a worktree path and remove it
   - Safety check: refuses to remove non-sprout worktrees (not under `~/.sprout`)

3. **Branch** (`sprout remove <branch>`)
   - Compute path: `$HOME/.sprout/<repo-slug>-<repo-id>/<branch>/<repo-slug>`
   - Remove that worktree if it exists

**Behavior:**

1. Validate the path is a sprout-managed worktree (under `~/.sprout`)
2. Remove the worktree via `git worktree remove`
3. Automatically run `git worktree prune` to clean up stale references

**Flags:**
- `--force`: Force removal even if the worktree has uncommitted changes

**Notes:**
- If the path is not a known worktree, sprout fails with a clear error message
- sprout refuses to remove worktrees that aren't managed by sprout (safety feature)

⸻

### 4. sprout list [--all]

List sprout-managed worktrees with git status indicators.

**Modes:**

1. **Current Repository** (`sprout list`)
   - Lists worktrees for the current repository only
   - Must be run from within a git repository

2. **All Repositories** (`sprout list --all`)
   - Lists worktrees from all sprout-managed repositories
   - Can be run from anywhere (doesn't require being in a git repo)
   - Groups worktrees by repository

**Behavior:**

- Scans filesystem for sprout-managed worktrees in all sprout directories
- Excludes the main worktree from output
- Displays git status indicators for each worktree:
  - 🔴 Dirty - worktree has uncommitted changes (via `git status --porcelain`)
  - ⬆️ Ahead - worktree has unpushed commits (via `git rev-list --count HEAD...@{upstream}`)
  - ⬇️ Behind - worktree needs to pull (via `git rev-list --count HEAD...@{upstream}`)
  - 🔀 Unmerged - worktree has commits not in main/master branch
- Pretty-prints with color styling and aligned columns

**Current repository output:**
```
abc                 🔴     /Users/you/.sprout/repo-a1b2c3d4/abc/repo
feature/new-feature        /Users/you/.sprout/repo-a1b2c3d4/feature/new-feature/repo
```

**All repositories output (`--all`):**
```
📦 my-repo                                    /Users/you/projects/my-repo
  abc                                   🔴    /Users/you/.sprout/my-repo-a1b2c3d4/abc/my-repo
  feature/new-feature                   🔀    /Users/you/.sprout/my-repo-a1b2c3d4/feature/new-feature/my-repo

📦 another-repo                               /Users/you/projects/another-repo
  main-dev                              🔴⬆️  /Users/you/.sprout/another-repo-e5f6g7h8/main-dev/another-repo
```

**Styling:**
- Green text for branch names
- Emoji status indicators in a separate aligned column
- Dim gray text for paths
- Bold repository names with 📦 emoji for `--all` mode
- Globally aligned columns for easy scanning
- Status column only appears if at least one worktree has status indicators

**Status Indicator Logic:**
- **Dirty**: Checked via `git status --porcelain` (non-empty output = dirty)
- **Ahead/Behind**: Checked via `git rev-list --left-right --count HEAD...@{upstream}` (requires upstream tracking)
- **Unmerged**: Checked via `git rev-list --count origin/main..HEAD` (or origin/master as fallback)
- All git operations are non-fatal; errors are silently skipped to avoid breaking the list output

**Flags:**
- `--all`: List worktrees from all repositories

**Notes:**
- Only shows worktrees that actually exist on the filesystem
- Scans the sprout root directory (`$XDG_DATA_HOME/sprout` or `~/.local/share/sprout`)
- Main worktree is intentionally excluded from the list
- Handles stale git metadata gracefully by scanning filesystem directly
- Multiple status indicators can appear together (e.g., 🔴🔀)
- Clean worktrees show no indicators

⸻

### 5. sprout trust [path]

Trust a repository to allow running hooks defined in `.sprout.yml`.

**Usage:**

```bash
# Trust current repository
sprout trust

# Trust specific repository
sprout trust /path/to/repo
```

**Behavior:**

1. If no path provided, uses the main worktree path of the current repository
2. Validates the path is a Git repository
3. Checks if already trusted
4. Adds the repository root path to `~/.config/sprout/trusted-projects.json`

**Security Model:**

Hooks can execute arbitrary commands on your system. sprout requires explicit trust before running any hooks to prevent malicious code execution from untrusted sources.

**⚠️ WARNING:** Only trust repositories you control or have reviewed the `.sprout.yml` file for.

**Trust Storage:**

Trusted repositories are stored in `~/.config/sprout/trusted-projects.json`:

```json
{
  "version": 1,
  "trusted": [
    {
      "repo_root": "/Users/you/projects/my-repo",
      "trusted_at": "2025-12-12T21:15:00Z"
    }
  ]
}
```

⸻

### 6. sprout hooks

Display hook configuration status for the current repository.

**Usage:**

```bash
sprout hooks
```

**Output includes:**

- Whether `.sprout.yml` exists and its location
- Trust status of the repository
- List of defined `on_create` hooks
- List of defined `on_open` hooks
- Commands that will trigger hooks
- Instructions for trusting the repository if not trusted

**Example output:**

```
Repository: /Users/you/projects/my-repo

✅ Config file: /Users/you/projects/my-repo/.sprout.yml

✅ Repository is trusted

on_create hooks:
  1. npm ci
  2. npm run build

on_open hooks:
  1. npm run lint:types

Hooks run automatically when:
  - sprout add           (runs on_create)
  - sprout open          (runs on_open)

Use --no-hooks flag to skip automatic execution.
```

⸻

### 7. sprout repair [--prune]

Repair git metadata for moved or relocated worktrees.

**Usage:**

```bash
# Repair all repositories
sprout repair

# Repair and prune stale references
sprout repair --prune
```

**Behavior:**

1. Discovers all sprout-managed repositories by scanning sprout root directories
2. For each repository, runs `git worktree repair` to fix metadata for moved worktrees
3. Optionally runs `git worktree prune` to remove stale worktree references

**When to use:**

- After moving sprout directories (e.g., from `~/.sprout` to `~/.local/share/sprout`)
- When `git worktree list` shows incorrect paths for existing worktrees
- To clean up git metadata after manual worktree directory changes

**Discovery:**

The command scans the sprout root directory:
- `$XDG_DATA_HOME/sprout` (if `$XDG_DATA_HOME` is set)
- `~/.local/share/sprout` (default)

For each directory found, it walks the filesystem to discover valid git worktrees and groups them by their main repository.

**Output:**

```
Found 3 repository(ies) to repair...

📦 my-repo
   ✅ Repaired worktree metadata
   🧹 Pruned stale worktree references

📦 another-repo
   ✅ Repaired worktree metadata
   🧹 Pruned stale worktree references

Summary:
  ✅ Repaired: 2
  🧹 Pruned: 2
```

**Flags:**
- `--prune` / `-p`: Also prune stale worktree references after repair

**⚠️ Important:**

Always run `sprout repair` WITHOUT `--prune` first when dealing with moved worktrees. The repair command updates git's metadata to reflect the current worktree locations. Only use `--prune` after verifying the repair worked correctly, as pruning will permanently remove metadata for worktrees that git cannot find.

**Workflow for moved worktrees:**

1. Move sprout directory (e.g., `mv ~/.sprout ~/.local/share/sprout`)
2. Run `sprout repair` to update git metadata
3. Verify with `sprout list --all`
4. Optionally run `sprout repair --prune` to clean up truly deleted worktrees

⸻

## Editor integration

When opening a worktree (via `sprout open` or after `sprout add`), sprout attempts to open an editor in this order:

1. Use a configured editor command if present (future config, e.g. `$SPROUT_EDITOR` or config file)
2. Platform-aware defaults, e.g. on macOS:
   - `open -a "Cursor" <path>`
   - fallback: `cursor <path>`
   - fallback: `code <path>`
   - fallback: `open <path>`

All editor spawning is non-blocking from the CLI perspective.

⸻

## External dependencies

**Required:**
- Git must be available on PATH

**Interactive Selection:**
- sprout uses `github.com/ktr0731/go-fuzzyfinder` for interactive selection (embedded, no external fzf required)
- Provides fuzzy finding for branch and worktree selection

**Shell Completion:**
- Branch name completion available for `add`, `open`, `remove` commands
- Enable via: `sprout completion [bash|zsh|fish|powershell]`

⸻

## Implementation notes

**Language:** Go

**Key Technologies:**
- `os/exec` to invoke git commands
- Cobra for CLI structure and shell completion
- `github.com/ktr0731/go-fuzzyfinder` for interactive selection
- `gopkg.in/yaml.v3` for `.sprout.yml` parsing

**Architecture:**
- Resolve repo root with `git rev-parse --show-toplevel`
- Compute `repo-id` with SHA-1 truncated to 8 characters
- Carefully handle spaces in paths and branch names

**Commands:**
- `sprout add` - Create worktrees (with optional hooks)
- `sprout open` - Open worktrees (with optional hooks)
- `sprout remove` - Remove worktrees (automatically prunes stale references)
- `sprout list` - List sprout-managed worktrees with git status indicators
- `sprout repair` - Repair git metadata for moved worktrees
- `sprout trust` - Trust repositories for hook execution
- `sprout hooks` - Display hook configuration status
- `sprout completion` - Generate shell completion scripts

**Design principles:**
- Small, composable subcommand handlers
- Clear error messages
- Non-blocking editor launching
- Safe by default (trust model for hooks)

⸻

## Non-goals

- No custom Git plumbing beyond calling `git worktree` and basic git commands
- No inspection or manipulation of commits, diffs, or PRs
- No complex configuration files beyond `.sprout.yml` for hooks

⸻

## Future ideas

Potential enhancements not yet implemented:

**Configuration:**
- Global config file (`~/.config/sprout/config.yaml`) for:
  - Default base branch (e.g. `develop` instead of `main`)
  - Editor command override
  - Custom sprout root (override `~/.sprout`)

**Features:**
- PR integration (e.g. `sprout add --pr 123`)
- A TUI mode that shows worktrees and their status using a single terminal UI
- `.sprout.local.yml` for per-developer overrides (useful for local customizations)
- Per-hook timeouts
- Hooks for other lifecycle events (e.g., `on_remove`)
- OS-specific hooks
- Parallel hook execution
- Hook execution history and logs

