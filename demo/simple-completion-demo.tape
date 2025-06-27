# Simple completion demo for workspace-manager
Output simple-completion-demo.gif
Set Width 1200
Set Height 600
Set TypingSpeed 100ms

# Setup
Type "export PATH=$PWD:$PATH"
Enter
Sleep 300ms
Type "source <(wsm _carapace)"
Enter
Sleep 500ms

# Show workspace completion for info
Type "wsm info add-d"
Tab
Sleep 2s
Screenshot workspace-completion.txt

# Complete and run the command
Enter
Sleep 2s

# Show repo completion for add command
Type "wsm add add-dynamic-carapace-completion gl"
Tab  
Sleep 2s
Screenshot repo-completion.txt
Ctrl+C

# Final demo
Type "echo 'Completion demo complete!'"
Enter
