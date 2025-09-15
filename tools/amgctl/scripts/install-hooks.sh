#!/bin/bash

set -e

# Get the repository root
REPO_ROOT="$(git rev-parse --show-toplevel)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"
TOOL_DIR="$REPO_ROOT/tools/amgctl"

# Check if we're in the amgctl directory or repository root
if [[ "$(pwd)" != "$TOOL_DIR" ]] && [[ "$(pwd)" != "$REPO_ROOT" ]]; then
    echo "Error: This script should be run from either:"
    echo "  - Repository root: $REPO_ROOT"
    echo "  - Tool directory: $TOOL_DIR"
    echo "Current directory: $(pwd)"
    exit 1
fi

echo "Installing unified Git hooks for amg-utils repository..."

# Create the unified pre-commit hook that handles both amgctl and amg-qad
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash

# Unified pre-commit hook for amg-utils repository
# This hook runs linting and formatting checks on both amgctl and amg-qad code

set -e

# Get the repository root
REPO_ROOT="$(git rev-parse --show-toplevel)"
AMG_QAD_DIR="$REPO_ROOT/tools/amg-qad"
AMGCTL_DIR="$REPO_ROOT/tools/amgctl"

# Get list of changed files
CHANGED_FILES=$(git diff --cached --name-only)

# Check if any amg-qad files are being committed
QAD_CHANGED=false
if echo "$CHANGED_FILES" | grep -q "^tools/amg-qad/"; then
    QAD_CHANGED=true
fi

# Check if any amgctl files are being committed
AMGCTL_CHANGED=false
if echo "$CHANGED_FILES" | grep -q "^tools/amgctl/"; then
    AMGCTL_CHANGED=true
fi

# Run amg-qad checks if needed
if [ "$QAD_CHANGED" = true ]; then
    echo "🔍 Running amg-qad pre-commit checks..."
    
    # Change to amg-qad directory
    cd "$AMG_QAD_DIR"
    
    # Run linting checks
    echo "📝 Checking amg-qad code formatting..."
    if ! make fmt-check; then
        echo "❌ amg-qad code formatting check failed!"
        echo "💡 Run 'cd tools/amg-qad && make fmt' to fix formatting issues"
        exit 1
    fi
    
    echo "🔧 Running amg-qad go vet..."
    if ! make vet; then
        echo "❌ amg-qad go vet check failed!"
        exit 1
    fi
    
    echo "🧹 Running amg-qad golangci-lint..."
    if ! make lint 2>/dev/null; then
        echo "❌ amg-qad linting check failed!"
        echo "💡 Run 'cd tools/amg-qad && make fix' to fix auto-fixable issues"
        echo "💡 Or run 'cd tools/amg-qad && make lint-install' to install golangci-lint"
        exit 1
    fi
    
    echo "✅ All amg-qad pre-commit checks passed!"
    
    # Return to repo root
    cd "$REPO_ROOT"
fi

# Run amgctl checks if needed
if [ "$AMGCTL_CHANGED" = true ]; then
    echo "🔍 Running amgctl pre-commit checks..."
    
    # Change to amgctl directory
    cd "$AMGCTL_DIR"
    
    # Run linting checks
    echo "📝 Checking amgctl code formatting..."
    if ! make fmt-check; then
        echo "❌ amgctl code formatting check failed!"
        echo "💡 Run 'cd tools/amgctl && make fmt' to fix formatting issues"
        exit 1
    fi
    
    echo "🔧 Running amgctl go vet..."
    if ! make vet; then
        echo "❌ amgctl go vet check failed!"
        exit 1
    fi
    
    echo "🧹 Running amgctl golangci-lint..."
    if ! make lint 2>/dev/null; then
        echo "❌ amgctl linting check failed!"
        echo "💡 Run 'cd tools/amgctl && make fix' to fix auto-fixable issues"
        echo "💡 Or run 'cd tools/amgctl && make lint-install' to install golangci-lint"
        exit 1
    fi
    
    echo "✅ All amgctl pre-commit checks passed!"
    
    # Return to repo root
    cd "$REPO_ROOT"
fi

# If no tool files were changed, skip all checks
if [ "$QAD_CHANGED" = false ] && [ "$AMGCTL_CHANGED" = false ]; then
    echo "No amg-qad or amgctl files changed, skipping tool-specific hooks"
fi

echo "🎉 All pre-commit checks completed successfully!"
EOF

# Make the hook executable
chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Unified Git hooks installed successfully!"
echo ""
echo "The pre-commit hook will now:"
echo "  - Run amgctl checks when you commit changes to tools/amgctl/"
echo "  - Run amg-qad checks when you commit changes to tools/amg-qad/"
echo "  - Run both sets of checks if you commit changes to both tools"
echo "  - Skip all checks if you commit non-tool files"
echo ""
echo "For each tool, the hook will:"
echo "  - Check code formatting with 'make fmt-check'"
echo "  - Run go vet for common errors"
echo "  - Run golangci-lint for comprehensive linting"
echo ""
echo "To fix issues:"
echo "  - Run 'cd tools/amgctl && make fix' for amgctl issues"
echo "  - Run 'cd tools/amg-qad && make fix' for amg-qad issues"
echo "  - Run 'make lint-install' in the respective tool directory if needed"
echo ""
echo "To skip the hook (not recommended):"
echo "  - Use 'git commit --no-verify'"
