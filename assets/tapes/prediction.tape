# VHS Tape: Command Prediction Demo
# Demonstrates the generative command suggestion feature

Output assets/prediction.gif

# Terminal settings
Set FontSize 16
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 15

# Start bish shell with isolated config (no pre-authorized commands)
Type "HOME=/tmp/bish-demo bin/bish"
Enter
Sleep 500ms

# Demo 1: Git command prediction
Type "git co"
Sleep 2s
# Ghost text should appear showing predicted commit command

# Accept prediction with Right
Right
Sleep 3s

# Clear the line
Ctrl+C
Sleep 500ms

# Demo 2: Another prediction example
Type "npm i"
Sleep 2s
# Ghost text shows npm install prediction

# Accept with Right arrow
Right
Sleep 3s

# Clear
Ctrl+C
Sleep 500ms

# Demo 3: Show clearing ghost text with Ctrl+K
Type "ls "
Sleep 3s
# Ghost text appears

# Clear ghost text only (keep input)
Ctrl+K
Sleep 2s
Ctrl+C
Sleep 2s

# Exit
Type "exit"
Enter
Sleep 500ms
