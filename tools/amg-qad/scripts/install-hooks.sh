#!/bin/bash

set -e

# Get the repository root
REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"
TOOL_DIR="$REPO_ROOT/tools/amg-qad"

# Check if we're in the amg-qad directory or repository root
if [[ "$(pwd)" != "$TOOL_DIR" ]] && [[ "$(pwd)" != "$REPO_ROOT" ]]; then
    echo "Error: This script should be run from either:"
    echo "  - Repository root: $REPO_ROOT"
    echo "  - Tool directory: $TOOL_DIR"
    echo "Current directory: $(pwd)"
    exit 1
fi

echo "Installing Git hooks for amg-qad..."

# Create the pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash

# AMG-QAD pre-commit hook
# This hook runs linting and formatting checks on amg-qad code

set -e

# Get the repository root
REPO_ROOT="$(git rev-parse --show-toplevel)"
AMG_QAD_DIR="$REPO_ROOT/tools/amg-qad"

# Check if any amg-qad files are being committed
if ! git diff --cached --name-only | grep -q "^tools/amg-qad/"; then
    echo "No amg-qad files changed, skipping amg-qad hooks"
    exit 0
fi

echo "🔍 Running amg-qad pre-commit checks..."

# Change to amg-qad directory
cd "$AMG_QAD_DIR"

# Run linting checks
echo "📝 Checking code formatting..."
if ! make fmt-check; then
    echo "❌ Code formatting check failed!"
    echo "💡 Run 'make fmt' to fix formatting issues"
    exit 1
fi

echo "🔧 Running go vet..."
if ! make vet; then
    echo "❌ go vet check failed!"
    exit 1
fi

echo "🧹 Running golangci-lint..."
if ! make lint 2>/dev/null; then
    echo "❌ Linting check failed!"
    echo "💡 Run 'make fix' to fix auto-fixable issues"
    echo "💡 Or run 'make lint-install' to install golangci-lint"
    exit 1
fi

echo "✅ All amg-qad pre-commit checks passed!"
EOF

# Make the hook executable
chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will now:"
echo "  - Run when you commit changes to tools/amg-qad/"
echo "  - Check code formatting with 'make fmt-check'"
echo "  - Run go vet for common errors"
echo "  - Run golangci-lint for comprehensive linting"
echo ""
echo "To fix issues:"
echo "  - Run 'make fix' to auto-fix formatting and linting"
echo "  - Run 'make lint-install' to install golangci-lint if needed"
echo ""
echo "To skip the hook (not recommended):"
echo "  - Use 'git commit --no-verify'"
