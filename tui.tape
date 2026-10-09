# VHS tape: drive PostBoy to the target state (BODY tab filled, response received).
#
# Prerequisites:
#   1. Test server running on localhost:18080:
#      go run ./cmd/testserver &
#   2. VHS installed: go install github.com/charmbracelet/vhs@latest
#
# Usage:
#   vhs tui.tape
#
# Output: tui.gif

Output tui.gif

Set FontSize 14
Set Width 1120
Set Height 800
Set TypingSpeed 50ms

# Build the binary first
Type "go build -o postboy ./cmd/postboy"
Enter
Sleep 2s

# Confirm terminal has 50 rows
Type "tput lines"
Enter
Sleep 500ms

# Launch the app
Type "./postboy"
Enter
Sleep 3s

# Type the URL explicitly
Type "http://localhost:18080/api/issues"
Sleep 500ms

# Set method to POST (from keymap: CycleMethod = alt+m)
Alt+m
Sleep 300ms

# Navigate to BODY tab (URL -> tabs -> right)
Tab
Sleep 300ms
Tab
Sleep 300ms
Right
Sleep 300ms

# Focus body editor and type JSON
Tab
Sleep 300ms
Type '{"title":"CLI headless test runner fails in CI","labels":["bug","terminal"]}'
Sleep 500ms

# Send request (Alt+R)
Alt+r
Sleep 3s

# Capture final state
Sleep 1s
