# Sprout Architecture

## Overview

Sprout follows the **"Functional Core, Imperative Shell"** pattern, separating pure business logic from side effects. This architecture makes the codebase highly testable, maintainable, and enables powerful features like dry-run mode.

```
┌─────────────────────────────────────────────────┐
│              CLI Entry Points                    │
│              (cobra commands)                    │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│          Imperative Shell                        │
│   • Gathers inputs via Effects                   │
│   • Calls Functional Core planners               │
│   • Executes plans via executor                  │
│   • Handles errors and exit codes                │
└─────────────┬──────────────┬────────────────────┘
              │              │
              ▼              ▼
   ┌──────────────┐   ┌──────────────┐
   │ Functional   │   │   Effects    │
   │    Core      │   │  Interface   │
   │  (pure)      │   │   (I/O)      │
   └──────────────┘   └──────────────┘
```

## Core Principles

### 1. Functional Core (Pure Functions)

**Location:** `internal/core/`

All business logic is implemented as **pure functions**:
- Deterministic: same inputs → same outputs
- No side effects: no I/O, no mutations
- Easy to test: no mocks needed
- Easy to reason about: no hidden state

**Example:** Planning what actions to take

```go
// Pure function - no I/O, fully testable
func PlanTrustCommand(ctx TrustContext) Plan {
    if ctx.RepoRoot == "" {
        return errorPlan(errNoRepoRoot)
    }

    if ctx.AlreadyTrusted {
        return Plan{Actions: []Action{
            PrintMessage{Msg: msgRepoAlreadyTrusted(ctx.RepoRoot)},
        }}
    }

    return Plan{Actions: []Action{
        TrustRepo{RepoRoot: ctx.RepoRoot},
        PrintMessage{Msg: msgRepoTrusted(ctx.RepoRoot)},
    }}
}
```

### 2. Imperative Shell (Side Effects)

**Location:** `cmd/`

The shell coordinates I/O and executes plans:
- Gathers inputs using the Effects interface
- Builds contexts for planning
- Calls pure planning functions
- Executes the resulting plan
- Handles errors and exit codes

**Example:** Trust command handler

```go
func runTrustCommand(cmd *cobra.Command, args []string) error {
    fx := effects.NewRealEffects()

    // Build context (imperative - does I/O)
    ctx, err := BuildTrustContext(fx, pathArg)
    if err != nil {
        return err
    }

    // Plan (pure - no I/O)
    plan := core.PlanTrustCommand(ctx)

    // Execute (imperative - does I/O)
    if err := effects.ExecutePlan(plan, fx); err != nil {
        if code, ok := effects.IsExit(err); ok {
            os.Exit(code)
        }
        return err
    }
    return nil
}
```

## The Action/Plan Pattern

### Actions (Typed Sum Type)

Actions represent **what should happen**, not **how to do it**. Each action is a typed struct implementing the `Action` interface:

```go
// Action is implemented by all action types
type Action interface{ isAction() }

// Example action types
type PrintMessage struct{ Msg string }
type RunGitCommand struct{ Dir string; Args []string }
type TrustRepo struct{ RepoRoot string }
type Exit struct{ Code int }
```

**Benefits over `map[string]any`:**
- Compile-time type safety (no runtime panics)
- IDE autocomplete works perfectly
- Refactoring is safe (rename field → compiler shows all usages)
- Clear documentation (struct fields are self-documenting)

### Plans (Sequence of Actions)

A `Plan` is an ordered list of actions:

```go
type Plan struct {
    Actions []Action
}
```

**Example plan:**

```go
Plan{Actions: []Action{
    CreateDirectory{Path: "/worktrees/feature", Perm: 0755},
    RunGitCommand{Dir: "/repo", Args: []string{"worktree", "add", "/worktrees/feature", "-b", "feature"}},
    PrintMessage{Msg: "Worktree created"},
    OpenEditor{Path: "/worktrees/feature"},
}}
```

### Plan Execution

The executor interprets plans using type switches:

```go
func executeAction(action Action, fx Effects) error {
    switch a := action.(type) {
    case PrintMessage:
        fx.Print(a.Msg)
    case RunGitCommand:
        _, err := fx.RunGitCommand(a.Dir, a.Args...)
        return err
    case TrustRepo:
        return fx.TrustRepo(a.RepoRoot)
    case Exit:
        return ExitError{Code: a.Code}
    // ... other cases
    }
    return nil
}
```

## The Effects Interface

**Location:** `internal/effects/effects.go`

The Effects interface abstracts all side effects:

```go
type Effects interface {
    // Git operations
    GetRepoRoot() (string, error)
    ListWorktrees(repoRoot string) ([]git.Worktree, error)
    RunGitCommand(dir string, args ...string) (string, error)

    // File system
    FileExists(path string) bool
    MkdirAll(path string, perm os.FileMode) error

    // Config and trust
    LoadConfig(currentPath, mainPath string) (*config.Config, error)
    IsTrusted(repoRoot string) (bool, error)
    TrustRepo(repoRoot string) error

    // Editor and output
    OpenEditor(path string) error
    Print(msg string)
    PrintErr(msg string)

    // Interactive selection
    SelectBranch(branches []git.Branch) (int, error)
    SelectWorktree(worktrees []git.Worktree) (int, error)
}
```

### Why Effects?

1. **Testability:** Swap real I/O with test doubles
2. **Clarity:** All side effects are explicit at function boundaries
3. **Flexibility:** Easy to add logging, metrics, or retry logic
4. **Predictability:** Pure functions can't do hidden I/O

### Two Implementations

**RealEffects** (`internal/effects/real.go`):
- Production implementation
- Delegates to actual git, filesystem, editor, etc.
- Used by CLI commands

**TestEffects** (`internal/effects/testeffects.go`):
- Test implementation
- Records all calls and returns predefined values
- No real I/O - fully in-memory
- Used by tests

## Testing Strategy

### 1. Pure Function Tests (Core)

Test planning logic without any I/O:

```go
func TestPlanTrustCommand_NotYetTrusted(t *testing.T) {
    ctx := core.TrustContext{
        RepoRoot:       "/test/repo",
        AlreadyTrusted: false,
    }

    plan := core.PlanTrustCommand(ctx)

    // Assert plan structure
    require.Len(t, plan.Actions, 2)

    // Type-safe assertions
    trustAction := plan.Actions[0].(core.TrustRepo)
    assert.Equal(t, "/test/repo", trustAction.RepoRoot)

    msgAction := plan.Actions[1].(core.PrintMessage)
    assert.Contains(t, msgAction.Msg, "trusted")
}
```

**Advantages:**
- Fast (no I/O)
- Reliable (no flaky filesystem/git dependencies)
- Focused (tests business logic only)

### 2. Context Builder Tests (Shell)

Test input gathering with TestEffects:

```go
func TestBuildTrustContext_CurrentRepo(t *testing.T) {
    fx := effects.NewTestEffects()
    fx.RepoRoot = "/test/repo"
    fx.TrustedRepos["/test/repo"] = true

    ctx, err := BuildTrustContext(fx, "")

    require.NoError(t, err)
    assert.Equal(t, "/test/repo", ctx.RepoRoot)
    assert.True(t, ctx.AlreadyTrusted)

    // Verify Effects were called correctly
    assert.Equal(t, 1, fx.GetMainWorktreePathCalls)
    assert.Equal(t, 1, fx.IsTrustedCalls)
}
```

**Advantages:**
- Tests real handler code (not reimplemented logic)
- No real filesystem/git operations
- Catches wiring bugs

### 3. End-to-End Tests

Test full flow: build context → plan → execute:

```go
func TestTrustCommand_EndToEnd(t *testing.T) {
    fx := effects.NewTestEffects()
    fx.RepoRoot = "/test/repo"
    fx.TrustedRepos["/test/repo"] = false

    // Full flow
    ctx, err := BuildTrustContext(fx, "")
    require.NoError(t, err)

    plan := core.PlanTrustCommand(ctx)
    err = effects.ExecutePlan(plan, fx)

    // Verify behavioral outcomes
    require.NoError(t, err)
    assert.True(t, fx.TrustedRepos["/test/repo"], "repo should be trusted")
    assert.Contains(t, fx.PrintedMsgs[0], "trusted")
}
```

**Advantages:**
- Tests all layers together
- Verifies behavioral outcomes (state changes)
- Still fast (uses TestEffects)

## Key Patterns

### 1. Context Structs

Each command has a context struct containing all inputs:

```go
type AddContext struct {
    Branch             string
    RepoRoot           string
    MainWorktreePath   string
    WorktreePath       string
    WorktreeExists     bool
    LocalBranchExists  bool
    RemoteBranchExists bool
    Config             *config.Config
    IsTrusted          bool
    NoHooks            bool
    NoOpen             bool
}
```

**Benefits:**
- Explicit dependencies
- Easy to test (just construct the struct)
- Self-documenting (field names explain what's needed)

### 2. Error Plans

Instead of mixing errors with side effects, return error plans:

```go
func errorPlan(msg string) Plan {
    return Plan{Actions: []Action{
        PrintError{Msg: msg},
        Exit{Code: 1},
    }}
}
```

### 3. Structured Tracking in Tests

TestEffects uses structured counters, not strings:

```go
type TestEffects struct {
    // Counters
    GetRepoRootCalls int
    TrustRepoCalls   int

    // Captured data
    PrintedMsgs []string
    GitCommands []GitCmd

    // Predefined responses
    RepoRoot      string
    TrustedRepos  map[string]bool
}
```

**Better than `Calls []string`:**
- Type-safe
- No string parsing
- Clear intent

## Dry-Run Mode

The action/plan pattern makes dry-run trivial:

```go
// In command handler
plan := core.PlanAddCommand(ctx)

if dryRunFlag {
    fmt.Println(core.FormatPlan(plan))
    return nil
}

return effects.ExecutePlan(plan, fx)
```

**FormatPlan** converts actions to human-readable descriptions:

```
Planned actions:
  1. Create directory: /worktrees/feature
  2. Run git command in /repo: git worktree add /worktrees/feature -b feature
  3. Print: "✨ Worktree created successfully"
  4. Open editor: /worktrees/feature
```

**Benefits:**
- Shows exactly what would happen
- No special dry-run logic in planners
- Demonstrates separation of planning from execution

## Command Patterns

### State-Mutating Commands (trust, add, open, remove)

Use the full action/plan pattern:

1. **Shell builds context** (imperative, uses Effects)
2. **Core plans actions** (pure function)
3. **Executor runs plan** (imperative, uses Effects)

### Display Commands (list)

Extract pure formatting functions, keep I/O in shell:

```go
// Pure formatter
func FormatWorktree(display WorktreeDisplay) string {
    emojis := BuildStatusEmojis(display.Status)
    path := ShortenPath(display.Path)
    return fmt.Sprintf("%s %s\n  %s", emojis, display.Branch, path)
}

// Shell does I/O and calls formatter
for _, wt := range worktrees {
    status := getWorktreeStatus(wt) // I/O here
    display := WorktreeDisplay{...}
    fmt.Println(core.FormatWorktree(display))
}
```

## Benefits of This Architecture

### Testability

- Pure functions: no mocks needed
- Effects interface: swap I/O implementations
- Fast tests: no real git/filesystem operations
- Comprehensive coverage: easy to test edge cases

### Maintainability

- Clear boundaries: core vs shell
- Explicit dependencies: no hidden global state
- Type safety: compile-time guarantees
- Refactoring: change core logic without touching I/O

### Features

- Dry-run mode: free from action/plan pattern
- Future: transaction logs, undo, remote execution
- Debugging: inspect plans before execution

### Reliability

- Deterministic core: same inputs → same outputs
- Isolated I/O: failures contained in shell
- Validated inputs: planners reject invalid contexts
- Fail-fast: validation before side effects

## Migration Path

The refactor followed this incremental approach:

1. **Phase 1:** Extract pure functions (git command building, filtering)
2. **Phase 2:** Create Action/Plan types and Effects interface
3. **Phase 3:** Refactor trust command (establish pattern)
4. **Phase 4:** Refactor add command (more complex, with tests)
5. **Phase 5:** Refactor remaining commands (open, remove, list, repair)
6. **Phase 6:** Add dry-run support and documentation

Each phase maintained backward compatibility and all tests passed.

## Common Questions

### Q: Why not use the action pattern for list/hooks commands?

**A:** They're pure display operations with no state mutations. The action pattern is designed for commands that **do things**. For read-only queries, extracting pure formatters is sufficient.

### Q: Why not use generics or Result types?

**A:** Go's idiomatic `(value, error)` pattern works well. Adding `Result[T]` would increase complexity without clear benefits. YAGNI (You Aren't Gonna Need It).

### Q: Why not split Effects into smaller interfaces?

**A:** Current single interface works fine. TestEffects can use anonymous structs to implement subsets if needed. We can split later if pain emerges (YAGNI again).

### Q: What about performance?

**A:** Negligible overhead. Planning is cheap (just building structs), and execution matches original imperative code. The separation of concerns actually enables future optimizations (parallel execution, caching).

## Further Reading

- [Functional Core, Imperative Shell](https://www.destroyallsoftware.com/screencasts/catalog/functional-core-imperative-shell) - Gary Bernhardt
- [Parse, don't validate](https://lexi-lambda.github.io/blog/2019/11/05/parse-don-t-validate/) - Related concept about types
- [Boundaries](https://www.destroyallsoftware.com/talks/boundaries) - Gary Bernhardt's talk on architecture

## See Also

- [`internal/core/`](../internal/core/) - Functional core implementation
- [`internal/effects/`](../internal/effects/) - Effects interface and implementations
- [`cmd/`](../cmd/) - Imperative shell (command handlers)
- [Refactor plan](.cursor/plans/fp_refactor_plan_35610cc2.plan.md) - Detailed migration history

