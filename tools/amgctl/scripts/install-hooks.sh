#!/bin/bash

#
# Git Hooks Installation Script for amgctl
#
# This script installs Git hooks from the repository into the local .git/hooks directory.
# It should be run by developers after cloning the repository.
#

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🪝 Installing Git hooks for amgctl...${NC}"

# Get the repository root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo "")

if [ -z "$REPO_ROOT" ]; then
    echo -e "${RED}❌ Error: Not in a Git repository${NC}"
    exit 1
fi

# Define paths
HOOKS_SOURCE_DIR="$REPO_ROOT/tools/amgctl/scripts/hooks"
HOOKS_TARGET_DIR="$REPO_ROOT/.git/hooks"

# Check if source hooks directory exists
if [ ! -d "$HOOKS_SOURCE_DIR" ]; then
    echo -e "${RED}❌ Error: Hooks source directory not found at $HOOKS_SOURCE_DIR${NC}"
    exit 1
fi

# Check if target hooks directory exists
if [ ! -d "$HOOKS_TARGET_DIR" ]; then
    echo -e "${RED}❌ Error: Git hooks directory not found at $HOOKS_TARGET_DIR${NC}"
    echo -e "${YELLOW}💡 Make sure you're in a Git repository${NC}"
    exit 1
fi

# Install each hook
HOOKS_INSTALLED=0
HOOKS_UPDATED=0

for hook_file in "$HOOKS_SOURCE_DIR"/*; do
    if [ -f "$hook_file" ]; then
        hook_name=$(basename "$hook_file")
        target_path="$HOOKS_TARGET_DIR/$hook_name"
        
        # Check if hook already exists
        if [ -f "$target_path" ]; then
            echo -e "${YELLOW}⚠️  Hook '$hook_name' already exists${NC}"
            
            # Compare files to see if update is needed
            if ! cmp -s "$hook_file" "$target_path"; then
                echo -e "${BLUE}📝 Updating '$hook_name'...${NC}"
                cp "$hook_file" "$target_path"
                chmod +x "$target_path"
                HOOKS_UPDATED=$((HOOKS_UPDATED + 1))
                echo -e "${GREEN}✅ Updated '$hook_name'${NC}"
            else
                echo -e "${GREEN}✅ '$hook_name' is already up to date${NC}"
            fi
        else
            echo -e "${BLUE}📝 Installing '$hook_name'...${NC}"
            cp "$hook_file" "$target_path"
            chmod +x "$target_path"
            HOOKS_INSTALLED=$((HOOKS_INSTALLED + 1))
            echo -e "${GREEN}✅ Installed '$hook_name'${NC}"
        fi
    fi
done

# Summary
echo ""
if [ $HOOKS_INSTALLED -gt 0 ] || [ $HOOKS_UPDATED -gt 0 ]; then
    echo -e "${GREEN}🎉 Git hooks installation completed!${NC}"
    [ $HOOKS_INSTALLED -gt 0 ] && echo -e "${GREEN}   📦 Installed: $HOOKS_INSTALLED hook(s)${NC}"
    [ $HOOKS_UPDATED -gt 0 ] && echo -e "${GREEN}   🔄 Updated: $HOOKS_UPDATED hook(s)${NC}"
else
    echo -e "${GREEN}✅ All hooks are already installed and up to date${NC}"
fi

echo ""
echo -e "${BLUE}ℹ️  Installed hooks:${NC}"
for hook_file in "$HOOKS_SOURCE_DIR"/*; do
    if [ -f "$hook_file" ]; then
        hook_name=$(basename "$hook_file")
        echo -e "   • ${GREEN}$hook_name${NC}"
    fi
done

echo ""
echo -e "${YELLOW}💡 These hooks will now run automatically on Git operations${NC}"
echo -e "${YELLOW}   To skip hooks temporarily: git commit --no-verify${NC}"
