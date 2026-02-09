#!/bin/bash
# install.sh - Installation script for navi (claude-sessions).
# Creates directories, installs hooks, merges settings, and builds the binary.

set -e

echo "╭──────────────────────────────────────╮"
echo "│  navi installer                      │"
echo "╰──────────────────────────────────────╯"
echo

# Constants
SESSIONS_DIR="$HOME/.claude-sessions"
HOOKS_DIR="$SESSIONS_DIR/hooks"

# Create directories
mkdir -p "$SESSIONS_DIR"
echo "✓ Created $SESSIONS_DIR"

# Get script directory (where install.sh is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$HOOKS_DIR"
echo "✓ Created $HOOKS_DIR"

# Copy hook script
if [ -f "$SCRIPT_DIR/hooks/notify.sh" ]; then
    cp "$SCRIPT_DIR/hooks/notify.sh" "$HOOKS_DIR/"
    chmod +x "$HOOKS_DIR/notify.sh"
    echo "✓ Installed notify.sh hook"
else
    echo "✗ Error: hooks/notify.sh not found in $SCRIPT_DIR"
    exit 1
fi

# Copy tool-tracker hook script
if [ -f "$SCRIPT_DIR/hooks/tool-tracker.sh" ]; then
    cp "$SCRIPT_DIR/hooks/tool-tracker.sh" "$HOOKS_DIR/"
    chmod +x "$HOOKS_DIR/tool-tracker.sh"
    echo "✓ Installed tool-tracker.sh hook"
else
    echo "✗ Error: hooks/tool-tracker.sh not found in $SCRIPT_DIR"
    exit 1
fi

# Claude Code settings configuration
CLAUDE_CONFIG_DIR="$HOME/.claude"
SETTINGS_FILE="$CLAUDE_CONFIG_DIR/settings.json"
NAVI_CONFIG_FILE="$HOOKS_DIR/config.json"

# Hook types that navi uses
HOOK_TYPES=("UserPromptSubmit" "Stop" "PermissionRequest" "SessionEnd" "PostToolUse" "SubagentStart" "SubagentStop" "TeammateIdle" "TaskCompleted")

# Read hook config from hooks/config.json shipped with navi
HOOK_CONFIG=$(cat "$SCRIPT_DIR/hooks/config.json")

# Function to save navi config for manual merging
save_for_manual() {
    cp "$SCRIPT_DIR/hooks/config.json" "$NAVI_CONFIG_FILE"
    echo ""
    echo "✓ Navi hook config saved to: $NAVI_CONFIG_FILE"
    echo ""
    echo "To manually add navi hooks, merge the contents of that file into:"
    echo "  $SETTINGS_FILE"
    echo ""
    echo "You can append navi hooks to your existing hook arrays, or replace them."
}

# Function to merge hooks into settings.json
merge_hooks() {
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo "⚠ jq not found. Please install jq for automatic configuration."
        save_for_manual
        return 0
    fi

    # Create claude config directory if it doesn't exist
    mkdir -p "$CLAUDE_CONFIG_DIR"

    # If no settings file exists, create new one with hook config
    if [ ! -f "$SETTINGS_FILE" ]; then
        echo "$HOOK_CONFIG" | jq '.' > "$SETTINGS_FILE"
        echo "✓ Created settings.json with navi hooks"
        return 0
    fi

    # Check for conflicts with existing hooks
    local conflicts=()
    for hook_type in "${HOOK_TYPES[@]}"; do
        if jq -e ".hooks.$hook_type // empty | length > 0" "$SETTINGS_FILE" > /dev/null 2>&1; then
            conflicts+=("$hook_type")
        fi
    done

    # No conflicts - merge automatically
    if [ ${#conflicts[@]} -eq 0 ]; then
        jq -s '.[0] * .[1]' "$SETTINGS_FILE" <(echo "$HOOK_CONFIG") > "$SETTINGS_FILE.tmp"
        mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"
        echo "✓ Merged navi hooks into settings.json"
        return 0
    fi

    # Conflicts detected - prompt user
    echo ""
    echo "⚠ Existing hooks detected for: ${conflicts[*]}"
    echo ""
    echo "Your current hook configurations:"
    for hook_type in "${conflicts[@]}"; do
        echo "  $hook_type:"
        jq -r ".hooks.$hook_type | @json" "$SETTINGS_FILE" 2>/dev/null | sed 's/^/    /'
    done
    echo ""
    echo "Options:"
    echo "  1) Override - Replace ALL hooks with navi hooks only (backup saved)"
    echo "  2) Merge    - Add navi hooks alongside existing hooks"
    echo "  3) Skip     - Continue without modifying hooks"
    echo "  4) Manual   - Save navi config for manual merging"
    echo "  5) Abort    - Exit without changes"
    echo ""
    read -rp "Choose [1/2/3/4/5]: " choice

    case $choice in
        1)
            cp "$SETTINGS_FILE" "$SETTINGS_FILE.bak"
            echo "✓ Backup saved to $SETTINGS_FILE.bak"
            # Replace hooks entirely with navi's hooks
            jq --argjson hooks "$(echo "$HOOK_CONFIG" | jq '.hooks')" '.hooks = $hooks' "$SETTINGS_FILE" > "$SETTINGS_FILE.tmp"
            mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"
            echo "✓ Replaced all hooks with navi configuration"
            ;;
        2)
            cp "$SETTINGS_FILE" "$SETTINGS_FILE.bak"
            echo "✓ Backup saved to $SETTINGS_FILE.bak"
            # Merge navi hooks with existing
            jq -s '.[0] * .[1]' "$SETTINGS_FILE" <(echo "$HOOK_CONFIG") > "$SETTINGS_FILE.tmp"
            mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"
            echo "✓ Merged navi hooks with existing configuration"
            ;;
        3)
            echo "⊘ Skipping hooks configuration"
            ;;
        4)
            save_for_manual
            ;;
        5)
            echo "Aborted. No changes made to settings.json."
            return 0
            ;;
        *)
            echo "Invalid choice. No changes made to settings.json."
            return 0
            ;;
    esac
}

merge_hooks

# Build the binary
BINARY_NAME="navi"
INSTALL_DIR="$HOME/.local/bin"

echo ""
echo "Building $BINARY_NAME..."
cd "$SCRIPT_DIR"

if ! command -v go &> /dev/null; then
    echo "✗ Error: Go is not installed. Please install Go and try again."
    exit 1
fi

if go build -o "$BINARY_NAME" ./cmd/navi/; then
    echo "✓ Built $BINARY_NAME binary"
else
    echo "✗ Build failed"
    exit 1
fi

# Optional install to ~/.local/bin
echo ""
read -p "Install to $INSTALL_DIR? [Y/n] " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
    mkdir -p "$INSTALL_DIR"
    cp "$BINARY_NAME" "$INSTALL_DIR/"
    echo "✓ Installed to $INSTALL_DIR/$BINARY_NAME"

    # Check if in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        echo "⚠ Note: $INSTALL_DIR is not in your PATH"
        echo "  Add this to your shell config: export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
else
    echo "Binary available at: $SCRIPT_DIR/$BINARY_NAME"
fi

# Final summary
echo
echo "╭──────────────────────────────────────╮"
echo "│  Installation complete!              │"
echo "╰──────────────────────────────────────╯"
echo
echo "To use navi:"
echo "  1. Start tmux sessions for your projects"
echo "  2. Launch Claude Code in each session"
echo "  3. Run 'navi' to monitor them"
echo
echo "Example:"
echo "  tmux new-session -d -s myproject -c ~/projects/myproject"
echo "  tmux send-keys -t myproject 'claude' Enter"
echo "  navi"
echo
