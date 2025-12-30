# VHS Tape: Agent Feature Demo
# Demonstrates the AI agent capabilities including Magic Fix

Output assets/agent.gif

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

# Demo 1: Magic Fix - run a failing command
Type "ls --invalid-option"
Sleep 1.5s
Enter
Sleep 3s
# Command fails with error

# Use Magic Fix
Type "#?"
Sleep 1.5s
Enter
Sleep 10s
# Agent analyzes the error and suggests a fix

# Show the new editing capability - press 'e' to edit
# Or press 'y' to accept
Type "y"
Enter
Sleep 5s

# Demo 2: Agent chat - ask a question that triggers permission prompt
Type "# check disk usage in this directory"
Enter
Sleep 8s
# Agent will attempt to run du command

# Permission prompt appears - show manage menu
Type "m"
Sleep 2s
Enter
Sleep 3s
# Interactive permission menu shown

# Toggle a permission and apply
Type " "
Sleep 2s
Enter
Sleep 4s

# Demo 3: Show agent controls
Type "#!tokens"
Enter
Sleep 4s
# Shows token usage stats

# Demo 4: Show help
Type "#!help"
Sleep 2s
Enter
Sleep 5s

# Exit
Type "exit"
Enter
Sleep 500ms
