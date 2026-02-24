Output carousel_demo.gif

Set Shell "zsh"
Set TypingSpeed 10ms
Set FontSize 14
Set FontFamily "MesloLGS NF"
Set Width 1200
Set Height 520
Set Padding 20
Set BorderRadius 8
Set WindowBar Colorful

Hide
Type "source ~/.zshrc && export PATH=/Users/dayna.blackwell/code/shelfctl/bin:$PATH"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms

# Open hub
Type "shelfctl"
Enter
Sleep 4000ms

# Navigate to Edit Book (3 downs from top of hub)
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Enter
Sleep 1500ms

# Multi-select 6 books
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Enter
Sleep 2000ms
Show

# ── Card 1: left ghost visible ────────────────────────────────────────────────
Sleep 2500ms

# ── Card 2: both peeks visible ────────────────────────────────────────────────
Right
Sleep 700ms

# ── Card 3: both peeks visible ────────────────────────────────────────────────
Right
Sleep 700ms

# ── Card 4: both peeks visible ────────────────────────────────────────────────
Right
Sleep 600ms

# ── Card 5: both peeks visible ────────────────────────────────────────────────
Right
Sleep 600ms

# ── Card 6: right ghost visible ───────────────────────────────────────────────
Right
Sleep 1500ms

# ── Navigate back left ────────────────────────────────────────────────────────
Left
Sleep 500ms
Left
Sleep 500ms

# ── End at card 4: both peeks visible ─────────────────────────────────────────
Sleep 1500ms
