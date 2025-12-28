#!/bin/bash
# Test script to demonstrate the interactive trust prompt

set -e

echo "=== Interactive Trust Prompt Demo ==="
echo ""
echo "This script demonstrates the new UX for trusting repositories."
echo ""

# Create a test repository
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create a .sprout.yml with hooks
cat > .sprout.yml <<'EOF'
hooks:
  on_create:
    - echo "Installing dependencies..."
    - echo "Building project..."
EOF

git add .sprout.yml
git commit -m "Add sprout hooks"

echo "✓ Created test repository at: $TEST_DIR"
echo "✓ Added .sprout.yml with hooks"
echo ""
echo "Now try running:"
echo "  cd $TEST_DIR"
echo "  sprout add test-branch"
echo ""
echo "You should see:"
echo "  1. A prompt showing the hooks that would run"
echo "  2. A question asking if you trust the repository [y/N]"
echo "  3. If you answer 'y', the worktree is created immediately"
echo ""
echo "Alternatively, in a non-interactive environment (like CI), you'll see:"
echo "  - A clear error message showing what hooks would run"
echo "  - Instructions: 'sprout trust' or '--no-hooks'"
echo ""

