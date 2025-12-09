#!/bin/bash
set -e

# Cleanup
rm -rf /tmp/sprout-test
rm -rf ~/.sprout/sprout-test-*

# Setup dummy repo
mkdir -p /tmp/sprout-test
cd /tmp/sprout-test
git init
echo "Hello" > README.md
git add README.md
git commit -m "Initial commit"

# Build sprout (assuming it's already built in the current directory)
SPROUT_BIN=/Users/maartenvansteenkiste/Documents/Personal/Projects/sprout/sprout

# Test add
echo "Testing add..."
$SPROUT_BIN add feat/test
REPO_ID=$(git rev-parse --show-toplevel | tr -d '\n' | shasum | head -c 8)
WORKTREE_PATH="$HOME/.sprout/sprout-test-$REPO_ID/feat/test"
echo "Expected worktree path: $WORKTREE_PATH"

if [ ! -d "$WORKTREE_PATH" ]; then
    echo "Worktree directory not created at $WORKTREE_PATH"
    exit 1
fi

# Ensure no upstream is configured for the new branch
set +e
git rev-parse --abbrev-ref --symbolic-full-name feat/test@{upstream} >/dev/null 2>&1
UPSTREAM_STATUS=$?
set -e

if [ "$UPSTREAM_STATUS" -eq 0 ]; then
    echo "Upstream unexpectedly configured for feat/test"
    exit 1
else
    echo "Upstream correctly not configured for feat/test"
fi

# Test list
echo "Testing list..."
$SPROUT_BIN list

# Test open (non-interactive)
echo "Testing open..."
# We can't easily test editor opening in a script without mocking, but we can check if it runs without error
# $SPROUT_BIN open feat/test

# Test remove
echo "Testing remove..."
$SPROUT_BIN remove feat/test
if [ -d "$WORKTREE_PATH" ]; then
    echo "Worktree directory not removed"
    exit 1
fi

# Test prune
echo "Testing prune..."
$SPROUT_BIN prune

echo "All tests passed!"
