Here’s an updated spec.md you can drop in directly, with the ~/.sprout layout, repo slug + hash, and the final name sprout baked in.

# sprout – Git worktree helper

## Overview

**sprout** is a lightweight Go CLI tool for managing Git worktrees.

It provides an ergonomic interface for creating, opening, listing, and removing worktrees, with interactive selection via `fzf` and smart handling of remote branches.

sprout’s main goals:

- Make Git worktrees trivial to use in day-to-day development.
- Keep project directories clean by storing worktrees under `~/.sprout`.
- Offer a small, predictable command surface (`sprout add`, `sprout open`, `sprout remove`, …).
- Integrate nicely with editors (Cursor, VS Code, …) without being editor-specific.

---

## Worktree storage layout

sprout never creates worktrees as siblings of the main repo.
Instead, all worktrees live under a central root:

```text
$HOME/.sprout/

Within that, worktrees are grouped by repository identity:

$HOME/.sprout/<repo-slug>-<repo-id>/<branch-path>/

	•	repo-slug: basename of the repo root
	•	Example: repo at /Users/you/Projects/vl-widgets → repo-slug = vl-widgets
	•	repo-id: a short, stable identifier derived from the repo path
	•	Example: repo-id = sha1(<absolute-repo-root>)[:8]
	•	Ensures two different clones with the same name don’t collide
	•	branch-path: the Git branch name, used as a path
	•	Example: branch bugfix/handover-double-message
→ directory: bugfix/handover-double-message
	•	sprout must ensure intermediate directories exist (mkdir -p).

Examples

Repo root:

/Users/maarten/Documents/Projects/vl-widgets

Branch:

bugfix/handover-double-message

Possible layout:

$HOME/.sprout/
  vl-widgets-a1b2c3d4/
    bugfix/
      handover-double-message/
        # worktree files here

sprout never touches the user’s project folder structure beyond reading Git metadata.

⸻

Repo detection

For any command, sprout must:
	1.	Run from somewhere inside a Git working directory.
	2.	Resolve the repo root via:

git rev-parse --show-toplevel


	3.	Use that absolute path to compute:
	•	repo-slug = basename(repo-root)
	•	repo-id = sha1(repo-root)[:8]
	•	repo-root itself for Git commands that must run in the main worktree.

sprout should fail clearly if it’s not inside a Git repo.

⸻

Commands

1. sprout add <branch>

Create a new worktree for <branch> under ~/.sprout.

Behavior:
	1.	Determine:

repo-root      = git rev-parse --show-toplevel
repo-slug      = basename(repo-root)
repo-id        = sha1(repo-root)[:8]
worktree-root  = $HOME/.sprout/<repo-slug>-<repo-id>
worktree-path  = <worktree-root>/<branch>  # branch used as nested path


	2.	Check for .sprout.yml with on_create hooks:
	•	If hooks exist and --skip-hooks not set:
		•	Verify repository is trusted
		•	If not trusted, show helpful error message and exit
	•	If hooks exist and --skip-hooks is set, skip hook execution

	3.	If origin/<branch> exists (remote branch):

git -C <repo-root> worktree add <worktree-path> origin/<branch> -b <branch>


	4.	If it doesn't exist:
	•	Create local branch from origin/main (or configurable default):

git -C <repo-root> worktree add <worktree-path> -b <branch> origin/main


	5.	Run on_create hooks automatically if:
	•	.sprout.yml exists with on_create hooks
	•	Repository is trusted
	•	--skip-hooks flag not set

	6.	Open the worktree in an editor (unless --no-open is set).

Flags:
	•	--skip-hooks: Skip running on_create hooks even if .sprout.yml exists
	•	--no-open: Skip opening the worktree in an editor

Notes:
	•	sprout must create parent directories: mkdir -p "$worktree-root/<parent-of-branch-path>".
	•	If the worktree already exists at that path, sprout should detect it and either:
	•	refuse with a clear message, or
	•	offer an option to just open it.

⸻

2. sprout open [branch-or-path]

Open an existing worktree in an editor.

Modes:
	1.	Interactive (sprout open)
	•	Requires fzf.
	•	List all worktrees for the current repo (excluding the main worktree).
	•	Display something like:

<branch>                      <path>
bugfix/handover-double-message  /Users/.../.sprout/vl-widgets-a1b2c3d4/bugfix/handover-double-message
feat/new-widget                 /Users/.../.sprout/vl-widgets-a1b2c3d4/feat/new-widget


	•	Use fzf to pick one.
	•	Open the selected path in an editor.

	2.	Path (sprout open /some/path)
	•	If the argument is an existing directory, treat it as the worktree path and open it.
	3.	Branch (sprout open <branch>)
	•	Compute worktree-path = $HOME/.sprout/<repo-slug>-<repo-id>/<branch>.
	•	Open it if it exists; otherwise, show a helpful error ("no worktree for this branch").

Behavior:
	1.	Check for .sprout.yml with on_open hooks:
	•	If hooks exist and --no-hooks not set:
		•	Verify repository is trusted
		•	If not trusted, show helpful error message and exit
	•	If hooks exist and --no-hooks is set, skip hook execution

	2.	Open the worktree in an editor

	3.	Run on_open hooks automatically if:
	•	.sprout.yml exists with on_open hooks
	•	Repository is trusted
	•	--no-hooks flag not set

Flags:
	•	--no-hooks: Skip running on_open hooks even if .sprout.yml exists

⸻

3. sprout remove [branch-or-path]

Remove an existing worktree.

Modes:
	1.	Interactive (sprout remove)
	•	Requires fzf.
	•	Same listing mechanism as open.
	•	After selecting a worktree, run:

git -C <repo-root> worktree remove <path>


	•	Do not show/remove the main repo root.

	2.	Path (sprout remove /some/path)
	•	If argument is an existing directory, treat it as a worktree path and remove it.
	3.	Branch (sprout remove <branch>)
	•	Compute the path based on the repo + branch convention.
	•	Remove that worktree.

Notes:
	•	Consider a --force flag to pass --force to git worktree remove.
	•	If the path is not a known worktree, sprout should fail clearly.

⸻

4. sprout list

List worktrees for the current repo.
	•	Wraps git worktree list --porcelain and pretty-prints:

main      /path/to/main/repo
bugfix/...  /Users/.../.sprout/vl-widgets-a1b2c3d4/bugfix/handover-double-message
feat/...    /Users/.../.sprout/vl-widgets-a1b2c3d4/feat/new-widget


	•	Optionally support flags such as:
	•	--all (future: cross-repo listing)
	•	--raw to print raw Git output.

⸻

5. sprout prune

Clean up stale worktree references.
	•	Runs:

git -C <repo-root> worktree prune


	•	Optionally, sprout can later add logic to detect and suggest removing orphaned directories under ~/.sprout/<repo-slug>-<repo-id> that are no longer tracked by Git.

⸻

Editor integration

When opening a worktree (via sprout open or optionally after sprout add), sprout should attempt, in order:
	1.	Use a configured editor command if present (future config, e.g. $SPROUT_EDITOR or config file).
	2.	Platform-aware defaults, e.g. on macOS:
	•	open -a "Cursor" <path>
	•	fallback: cursor <path>
	•	fallback: code <path>
	•	fallback: open <path>

All editor spawning should be non-blocking from the CLI perspective.

⸻

External dependencies
	•	Git (obviously) must be available on PATH.

fzf shouldn't be an external dependency. It should be included in sprout.

⸻

Implementation notes
	•	Language: Go.
	•	Use os/exec to invoke git.
	•	Resolve repo root with git rev-parse --show-toplevel.
	•	Compute repo-id with a stable hash function (e.g. SHA-1 or SHA-256 truncated).
	•	Carefully handle spaces in paths and branch names (quote shell arguments correctly).

CLI structure:
	•	Either a minimal custom flag parser or a small framework like Cobra.
	•	Commands:
	•	sprout add
	•	sprout open
	•	sprout remove
	•	sprout list
	•	sprout prune

Aim for small, composable subcommand handlers that are easy to test.

⸻

Non-goals for v1
	•	No custom Git plumbing beyond calling git worktree and basic git commands.
	•	No inspection or manipulation of commits, diffs, or PRs.
	•	No global multi-repo dashboard (beyond possibly a simple list per repo).
	•	No persistent configuration beyond simple environment-based behavior.

⸻

Future ideas (out of scope for v1)
	•	Config file (~/.config/sprout/config.yaml) for:
	•	Default base branch (e.g. develop instead of main)
	•	Editor command
	•	Custom sprout root (override ~/.sprout)
	•	Cross-repo overview: sprout list --all (scan ~/.sprout and show everything).
	•	PR integration (e.g. sprout add --pr 123).
	•	A TUI mode that shows worktrees and their status using a single terminal UI.

