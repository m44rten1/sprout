package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/m44rten1/sprout/internal/core"
	"github.com/m44rten1/sprout/internal/effects"
)

func runAddContext(ctx core.AddContext, fx effects.Effects) {
	if !ctx.MoveCurrentChanges {
		runPlan(core.PlanAddCommand(ctx), fx)
		return
	}

	if dryRunFlag {
		fx.Print(formatMoveCurrentChangesPlan(ctx))
		return
	}

	if err := executeMoveCurrentChangesAdd(ctx, fx); err != nil {
		if code, ok := effects.IsExit(err); ok {
			os.Exit(code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func executeMoveCurrentChangesAdd(ctx core.AddContext, fx effects.Effects) error {
	if ctx.WorktreeExists {
		return fmt.Errorf("--move-current-changes requires a new worktree, but %s already exists", ctx.WorktreePath)
	}
	if !ctx.CurrentWorktreeDirty {
		return fmt.Errorf("--move-current-changes requested, but the current worktree has no uncommitted changes")
	}

	if shouldRunCreateHooks(ctx) && !ctx.IsTrusted {
		if err := fx.PromptTrustRepo(ctx.MainWorktreePath, string(core.HookTypeOnCreate), ctx.Config.Hooks.OnCreate); err != nil {
			return fmt.Errorf("prompt trust: %w", err)
		}
	}

	fx.Print(fmt.Sprintf("Stashing current changes from %s...", ctx.RepoRoot))
	stashOID, err := fx.CreateStash(ctx.RepoRoot, stashMessage(ctx.Branch), true)
	if err != nil {
		return fmt.Errorf("stash current changes: %w", err)
	}

	createCtx := createOnlyAddContext(ctx)
	if err := effects.ExecutePlan(core.PlanAddCommand(createCtx), fx); err != nil {
		return fmt.Errorf("create worktree: %w\n\nCurrent changes were preserved in stash %s.\nTo restore them in the original worktree, run:\n  git stash apply %s", err, stashOID, stashOID)
	}

	fx.Print(fmt.Sprintf("Applying stashed changes in %s...", ctx.WorktreePath))
	if err := fx.ApplyStash(ctx.WorktreePath, stashOID); err != nil {
		return fmt.Errorf("apply stashed changes in %s: %w\n\nYour changes are still preserved in stash %s.\nResolve the issue in the new worktree or recover manually with:\n  git -C %s stash apply %s", ctx.WorktreePath, err, stashOID, ctx.RepoRoot, stashOID)
	}

	if err := fx.DropStash(ctx.RepoRoot, stashOID); err != nil {
		return fmt.Errorf("drop temporary stash %s: %w\n\nYour changes were moved to %s, but the stash entry was kept for safety.\nYou can remove it manually with:\n  git -C %s stash drop %s", stashOID, err, ctx.WorktreePath, ctx.RepoRoot, stashOID)
	}

	if !ctx.NoOpen {
		if err := fx.OpenEditor(ctx.WorktreePath); err != nil {
			return fmt.Errorf("open editor for %s: %w", ctx.WorktreePath, err)
		}
	}

	if shouldRunCreateHooks(ctx) {
		if err := fx.RunHooks(ctx.RepoRoot, ctx.WorktreePath, ctx.MainWorktreePath, ctx.Config.Hooks.OnCreate, string(core.HookTypeOnCreate)); err != nil {
			return fmt.Errorf("run %s hooks: %w", core.HookTypeOnCreate, err)
		}
	}

	return nil
}

func createOnlyAddContext(ctx core.AddContext) core.AddContext {
	createCtx := ctx
	createCtx.NoOpen = true
	createCtx.NoHooks = true
	return createCtx
}

func shouldRunCreateHooks(ctx core.AddContext) bool {
	return ctx.Config.HasCreateHooks() && !ctx.NoHooks
}

func stashMessage(branch string) string {
	return fmt.Sprintf("sprout move-current-changes to %s", branch)
}

func formatMoveCurrentChangesPlan(ctx core.AddContext) string {
	lines := []string{
		"Planned actions:",
		fmt.Sprintf("  1. Stash current changes in %s (including untracked files)", ctx.RepoRoot),
		fmt.Sprintf("  2. Create worktree: %s", ctx.WorktreePath),
		fmt.Sprintf("  3. Apply the temporary stash in %s", ctx.WorktreePath),
		"  4. Drop the temporary stash after a successful apply",
	}

	nextStep := 5
	if !ctx.NoOpen {
		lines = append(lines, fmt.Sprintf("  %d. Open editor: %s", nextStep, ctx.WorktreePath))
		nextStep++
	}
	if shouldRunCreateHooks(ctx) {
		lines = append(lines, fmt.Sprintf("  %d. Run %d %s hook(s) in %s", nextStep, len(ctx.Config.Hooks.OnCreate), core.HookTypeOnCreate, ctx.WorktreePath))
	}

	return strings.Join(lines, "\n")
}
