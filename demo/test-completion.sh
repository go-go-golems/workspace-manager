#!/bin/bash

# Test carapace completion using tmux
set -e

DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DEMO_DIR"

# Start tmux session
SESSION_NAME="wsm-completion-test"
tmux new-session -d -s "$SESSION_NAME" -x 120 -y 30

# Setup environment in tmux
tmux send-keys -t "$SESSION_NAME" "export PATH=$DEMO_DIR:\$PATH" Enter
sleep 1
tmux send-keys -t "$SESSION_NAME" "source <(wsm _carapace)" Enter
sleep 1

echo "Testing workspace name completion for 'info' command..."
tmux send-keys -t "$SESSION_NAME" "wsm info " 
tmux send-keys -t "$SESSION_NAME" Tab
sleep 2
tmux capture-pane -t "$SESSION_NAME" -p > info-completion-output.txt
echo "Info completion captured"

# Clear the line and test tag completion
tmux send-keys -t "$SESSION_NAME" C-c
tmux send-keys -t "$SESSION_NAME" "wsm list repos --tags "
tmux send-keys -t "$SESSION_NAME" Tab
sleep 2
tmux capture-pane -t "$SESSION_NAME" -p > tag-completion-output.txt
echo "Tag completion captured"

# Clear and test add command workspace completion
tmux send-keys -t "$SESSION_NAME" C-c
tmux send-keys -t "$SESSION_NAME" "wsm add add-"
tmux send-keys -t "$SESSION_NAME" Tab
sleep 2
tmux capture-pane -t "$SESSION_NAME" -p > add-workspace-completion-output.txt
echo "Add workspace completion captured"

# Test repository completion for add command
tmux send-keys -t "$SESSION_NAME" "dynamic-carapace-completion "
tmux send-keys -t "$SESSION_NAME" "gla"
tmux send-keys -t "$SESSION_NAME" Tab
sleep 2
tmux capture-pane -t "$SESSION_NAME" -p > add-repo-completion-output.txt
echo "Add repository completion captured"

# Clear and test remove command repository completion
tmux send-keys -t "$SESSION_NAME" C-c
tmux send-keys -t "$SESSION_NAME" "wsm remove add-dynamic-carapace-completion "
tmux send-keys -t "$SESSION_NAME" Tab
sleep 2
tmux capture-pane -t "$SESSION_NAME" -p > remove-repo-completion-output.txt
echo "Remove repository completion captured"

# Clean up
tmux kill-session -t "$SESSION_NAME"

echo "Completion test complete. Check the *-output.txt files for results."
echo "Generated files:"
ls -la *-output.txt
