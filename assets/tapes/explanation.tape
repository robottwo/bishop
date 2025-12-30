# VHS Tape: Command Explanation Demo
# Demonstrates the automatic command explanation feature

Output assets/explanation.gif

# Terminal settings
Set FontSize 16
Set Width 900
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 15

# Start bish shell with isolated config (no pre-authorized commands)
Type "HOME=/tmp/bish-demo bin/bish"
Enter
Sleep 500ms

# Demo 1: Explain a tar command with flags
Type "tar -xzf archive.tar.gz"
Sleep 3s
# Explanation should appear in the assistant box at bottom
# explaining what -x, -z, -f flags do

Sleep 3s

# Clear
Ctrl+C
Sleep 2s

Type "BISH_ASSISTANT_HEIGHT=8"
Sleep 2s
Enter
Sleep 1.5s

# Demo 2: Git command with interactive flag
Type "git rebase -i HEAD~3"
Sleep 3s
# Explanation shows what interactive rebase does

Sleep 3s

# Clear
Ctrl+C
Sleep 500ms

# Demo 3: rsync with multiple flags
Type "rsync -avz --progress src/ dest/"
Sleep 3s
# Detailed explanation of flags

Sleep 3s

# Exit
Type "exit"
Enter
Sleep 1000ms
