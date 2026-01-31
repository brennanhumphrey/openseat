# VT Sniper Demo Recording
# Run from project root: vhs demo.tape

Output assets/demo.gif

# Terminal settings
Set Shell "zsh"
Set FontSize 13
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 700
Set Height 820
Set Padding 16
Set BorderRadius 8
Set WindowBar Colorful
Set Theme "GruvboxDark"
Set TypingSpeed 75ms
Set Framerate 30
Set LoopOffset 70%
Set Margin 12
Set MarginFill "#282828"

# Start with a clean prompt
Sleep 300ms

# Type and run the command
Type "./vt-sniper --demo"
Sleep 400ms
Enter

# Demo mode runs automatically:
# - Shows banner + config
# - Fetches 2 courses
# - Attempt 1: checks both, no seats
# - Attempt 2: finds DSA, sends email
# - Attempt 3: finds Computer Systems, sends email
# - Shows "All courses found!" and exits

# Wait for the entire demo to complete (~25 seconds)
Sleep 25s

# Brief pause at the end
Sleep 2s
