# VHS Tape: Agent Coding Demo
# Demonstrates the agent's ability to edit files and perform coding tasks

Output assets/agent_coding.gif

# Terminal settings
Set FontSize 16
Set Width 1000
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 15

# Start bish shell with isolated config (no pre-authorized commands)
Type "HOME=/tmp/bish-demo bin/bish"
Enter
Sleep 500ms

# Demo 1: Ask agent to create/edit code
Type "# add a comment to the main function explaining what it does"
Enter
Sleep 9s
# Agent proposes an edit with diff preview

# Approve the edit
Type "y"
Enter
Sleep 4s

# Demo 2: Use git diff macro
Type "#/gitdiff"
Enter
Sleep 4s
# Agent summarizes the current git changes

# Demo 3: Ask agent to explore code
Type "# what does the shell loop do in this codebase?"
Enter
Sleep 8s
# Agent uses view_file tool to explore

# Approve file read if needed
Type "y"
Enter
Sleep 3s

# Demo 4: Git push macro workflow
Type "#/gitpush"
Enter
Sleep 3s
# Agent helps with commit workflow

# Cancel if needed
Type "n"
Enter
Sleep 1s

# Exit
Type "exit"
Enter
Sleep 500ms
