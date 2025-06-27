# VHS demo for workspace-manager carapace completion
Output completion-demo.gif
Set Width 1200
Set Height 600

# Add current directory to PATH so wsm can be found
Type "export PATH=$PWD:$PATH"
Enter
Sleep 300ms

# Source carapace completion for the demo binary
Type "source <(wsm _carapace)"
Enter
Sleep 500ms

# Show completion for info command (partial typing + tab shows workspace names)
Type "wsm info fin"
Tab
Sleep 1s
Screenshot info-completion.txt

# Show the completion working by completing the name
Backspace 3
Type "finish-filepicker"
Enter
Sleep 1s

# Clear and show list repos with tag completion
Ctrl+C
Type "wsm list repos --tags "
Tab
Sleep 1s  
Screenshot tag-completion.txt

# Clear and show add command completion for workspace names
Ctrl+C
Type "wsm add add-"
Tab
Sleep 1s
Screenshot add-workspace-completion.txt

# Show second argument completion for repositories
Space
Type "gla"
Tab
Sleep 1s
Screenshot add-repo-completion.txt

# Clear and show remove command completion  
Ctrl+C
Type "wsm remove add-dynamic-carapace-completion "
Tab
Sleep 1s
Screenshot remove-repo-completion.txt

# End demo
Ctrl+C
