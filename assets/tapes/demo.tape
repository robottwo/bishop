# VHS Tape: Main Demo
# End-to-end demonstration of bishop's key features

Output assets/demo.gif

# Terminal settings
Set FontSize 16
Set Width 1000
Set Height 900
Set Theme "Catppuccin Mocha"
Set Padding 20
Set Framerate 15

# Start bish shell with isolated config (no pre-authorized commands)
Type "HOME=/tmp/bish-demo bin/bish"
Enter
Sleep 500ms

# === Feature 1: Command Prediction + Explanation ===
Type "git sta"
Sleep 2.5s
# Ghost text appears with prediction, explanation in assistant box

Right
Sleep 2s
Enter
Sleep 2s

Type "BISH_ASSISTANT_HEIGHT=8"
Sleep 1.5s
Enter
Sleep 2s

# === Feature 2: Quick agent interaction ===
Type "# Summarize the previus command output"
Sleep 1s
Enter
Sleep 5s

# === Feature 3: Show prediction on another command ===
Type "make bu"
Sleep 1.5s
Right
Sleep 1s
Ctrl+C
Sleep 500ms

# === Feature 4: Magic Fix demo ===
Type "cat nonexistent-file.txt"
Enter
Sleep 3s

Type "#?"
Enter
Sleep 5s
Type "n"
Sleep 2s

# === Feature 5: Subagent discovery ===
Type "#"
Sleep 1
Tab
Sleep 5s
# Shows available subagents
Ctrl+C
Sleep 500ms
Type "#!subagents"
Sleep 2s
Enter
Sleep 5s

# === Feature 6: Coach stats ===
Type "#!coach stats"
Enter
Sleep 5s

# Exit
Type "exit"
Enter
Sleep 5s
