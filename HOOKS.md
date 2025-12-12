# Sprout Hooks

Project hooks allow you to automate setup and sync tasks for your worktrees. This is especially useful for ensuring worktrees are always ready to work with, running tasks like dependency installation, builds, type checking, and code generation.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Security Model](#security-model)
- [Commands](#commands)
- [Environment Variables](#environment-variables)
- [Example Configurations](#example-configurations)
- [Troubleshooting](#troubleshooting)

## Quick Start

1. Create a `.sprout.yml` file in your repository root:

```yaml
hooks:
  on_create:
    - npm ci
    - npm run build
  on_open:
    - npm run lint:types
```

2. Trust your repository:

```bash
sprout trust
```

3. Use hooks with sprout commands:

```bash
# Create a new worktree and run bootstrap
sprout add feat/new-feature --init

# Open a worktree and run sync hooks
sprout open --sync
```

## Configuration

### File Location

Hooks are configured in `.sprout.yml` in your repository root (git top-level directory).

### Schema

```yaml
hooks:
  on_create:
    - <command 1>
    - <command 2>
  on_open:
    - <command 1>
    - <command 2>
```

### Hook Types

#### `on_create`

Runs after creating a new worktree. Ideal for:
- Installing dependencies (`npm ci`, `go mod download`)
- Building the project (`npm run build`, `go build`)
- Database migrations
- Generating code or assets

**Triggered by:**
- `sprout add <branch> --init`
- `sprout init`

#### `on_open`

Runs when explicitly syncing a worktree. Ideal for:
- Type checking (`npm run lint:types`, `go vet`)
- Code generation (`npm run generate`, `go generate`)
- Lightweight sync operations
- Database migrations

**Triggered by:**
- `sprout open <branch> --sync`
- `sprout sync`

### Validation Rules

- `hooks` section is optional
- Each hook type is optional
- Commands must be non-empty strings
- Commands are executed sequentially
- If a command fails, subsequent commands are skipped

## Security Model

### Why Trust is Required

Hooks execute arbitrary shell commands on your system. To prevent malicious code execution, you must **explicitly trust** each repository before hooks will run.

### Trusting a Repository

```bash
# Trust current repository
sprout trust

# Trust a specific path
sprout trust /path/to/repo
```

### Trust Storage

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

### Best Practices

- **Review `.sprout.yml` before trusting** - Understand what commands will run
- Only trust repositories you control or from trusted sources
- Be cautious with repositories containing sensitive operations
- Regularly audit your trusted repositories

## Commands

### `sprout add --init`

Create a new worktree and run `on_create` hooks:

```bash
sprout add feat/new-feature --init
```

Your editor opens immediately, then hooks run in the terminal. This allows you to start browsing code while dependencies install and builds complete.

**Skip opening the editor:**
```bash
sprout add feat/new-feature --init --no-open
```

Useful for automation or CI/CD scenarios where you only want the worktree created and initialized.

### `sprout init`

Manually run `on_create` hooks in the current worktree:

```bash
cd ~/.sprout/my-project/feat/bug-fix
sprout init
```

Useful for:
- Recovering from a failed initial bootstrap
- Re-running setup after configuration changes

### `sprout open --sync`

Open a worktree and run `on_open` hooks:

```bash
sprout open feat/bug-fix --sync
```

Your editor opens immediately, then hooks run in the terminal. You can start working while type-checking and code generation complete in the background.

### `sprout sync`

Run `on_open` hooks in the current worktree:

```bash
cd ~/.sprout/my-project/feat/bug-fix
sprout sync
```

Useful for:
- Freshening up a worktree before working
- Running type checks and code generation
- Syncing after pulling changes

### `sprout trust`

Trust a repository to run hooks:

```bash
# Trust current repo
sprout trust

# Trust specific repo
sprout trust /path/to/repo
```

### `sprout hooks`

Display hook configuration status:

```bash
sprout hooks
```

Shows:
- Whether `.sprout.yml` exists
- Trust status
- Defined hooks
- Available commands

## Environment Variables

When hooks run, the following environment variables are set:

- `SPROUT_REPO_ROOT` - Path to the git repository root
- `SPROUT_WORKTREE_PATH` - Path to the current worktree
- `SPROUT_HOOK_TYPE` - Either `on_create` or `on_open`

### Example Usage

```yaml
hooks:
  on_create:
    - echo "Bootstrapping worktree at $SPROUT_WORKTREE_PATH"
    - echo "Repository root: $SPROUT_REPO_ROOT"
```

## Example Configurations

### Node.js Project

```yaml
hooks:
  on_create:
    - npm ci
    - npm run build
  on_open:
    - npm run lint:types
    - npm run generate
```

### Go Project

```yaml
hooks:
  on_create:
    - go mod download
    - go build
    - go generate ./...
  on_open:
    - go mod tidy
    - go vet ./...
```

### Python Project

```yaml
hooks:
  on_create:
    - python -m venv .venv
    - source .venv/bin/activate && pip install -r requirements.txt
  on_open:
    - source .venv/bin/activate && mypy .
```

### Monorepo with Multiple Tools

```yaml
hooks:
  on_create:
    - npm ci
    - npm run build:packages
    - npm run db:migrate
  on_open:
    - npm run lint:types
    - npm run generate:types
    - npm run db:migrate
```

### Conditional Commands

```yaml
hooks:
  on_create:
    - test -f package.json && npm ci || echo "No package.json"
    - test -f Gemfile && bundle install || echo "No Gemfile"
  on_open:
    - git fetch origin
```

## Troubleshooting

### Hooks Not Running

**Issue:** Commands don't execute when using `--init` or `--sync`.

**Solutions:**
1. Check if repository is trusted: `sprout hooks`
2. Trust the repository: `sprout trust`
3. Verify `.sprout.yml` exists in repo root
4. Check YAML syntax is valid

### Command Fails

**Issue:** A hook command exits with an error.

**Behavior:**
- Remaining commands in that hook are skipped
- Error message displays the failed command and exit code

**Solutions:**
1. Review command syntax in `.sprout.yml`
2. Ensure dependencies are available (npm, go, etc.)
3. Check environment variables are set correctly
4. Add error handling: `command || true` to continue on failure

### Config Parse Error

**Issue:** `failed to parse config file` error.

**Solutions:**
1. Validate YAML syntax at https://www.yamllint.com/
2. Ensure proper indentation (2 spaces per level)
3. Commands must be strings, not complex objects
4. Check for special characters that need escaping

### Shell Not Found

**Issue:** Commands fail with "command not found".

**Behavior:**
Commands run via `sh -lc "<command>"` which loads your shell profile.

**Solutions:**
1. Ensure commands are in your PATH
2. Use full paths: `/usr/local/bin/npm ci`
3. Check shell profile (`.profile`, `.zshrc`) for PATH setup

## Future Enhancements

Potential features not yet implemented:

- `.sprout.local.yml` for per-developer overrides
- Per-hook timeouts
- Hooks for other lifecycle events (e.g., `on_remove`)
- OS-specific hooks
- Parallel hook execution
- Hook execution history and logs

## Contributing

Found a bug or have a feature request? Open an issue at the Sprout repository.
