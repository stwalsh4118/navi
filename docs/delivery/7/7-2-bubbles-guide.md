# Bubbles TextInput Guide for Task 7-2

**Date**: 2026-02-05
**Package**: `github.com/charmbracelet/bubbles/textinput`
**Documentation**: [pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput)

## Overview

The `textinput` package provides a text input component for Bubble Tea applications. It supports unicode, pasting, in-place scrolling when the value exceeds the width, and many customization options.

## Installation

```bash
go get github.com/charmbracelet/bubbles
```

## Creating a New TextInput

```go
import "github.com/charmbracelet/bubbles/textinput"

// Create a new textinput model
input := textinput.New()
input.Placeholder = "Enter session name"
input.CharLimit = 50
input.Width = 40
input.Focus() // Give it focus initially
```

## Key Methods

### Value Management
- `Value() string` - Returns the current input value
- `SetValue(s string)` - Sets the input value
- `Reset()` - Resets the input to default state with no input

### Focus Management
- `Focus() tea.Cmd` - Sets focus (enables keyboard input, shows cursor)
- `Blur()` - Removes focus (disables keyboard input, hides cursor)
- `Focused() bool` - Returns focus state

### Update & Rendering
- `Update(msg tea.Msg) (Model, tea.Cmd)` - Bubble Tea update loop
- `View() string` - Renders the textinput in current state

## Customization Fields

```go
type Model struct {
    Prompt           string         // Text displayed before input
    Placeholder      string         // Text shown when empty
    CharLimit        int            // Max characters (0 = no limit)
    Width            int            // Visible width (0 = no limit)

    // Styling
    PromptStyle      lipgloss.Style
    TextStyle        lipgloss.Style
    PlaceholderStyle lipgloss.Style
}
```

## Usage in navi Project

### In Model struct

```go
type Model struct {
    // ... existing fields ...

    // Text inputs for dialogs
    nameInput textinput.Model
    dirInput  textinput.Model
}
```

### Initialization

```go
func initTextInputs() (textinput.Model, textinput.Model) {
    nameInput := textinput.New()
    nameInput.Placeholder = "Session name"
    nameInput.CharLimit = 50
    nameInput.Width = 40
    nameInput.Focus()

    dirInput := textinput.New()
    dirInput.Placeholder = "Working directory"
    dirInput.CharLimit = 256
    dirInput.Width = 40

    return nameInput, dirInput
}
```

### Handling Updates

```go
func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch m.dialogMode {
    case DialogNewSession:
        // Update the focused input
        m.nameInput, cmd = m.nameInput.Update(msg)
    }

    return m, cmd
}
```

### Rendering in Dialog

```go
func (m Model) renderDialog() string {
    // ...
    switch m.dialogMode {
    case DialogNewSession:
        b.WriteString("Name: ")
        b.WriteString(m.nameInput.View())
    }
    // ...
}
```

## Validation Pattern

```go
func validateSessionName(name string, existingSessions []SessionInfo) error {
    if name == "" {
        return errors.New("session name cannot be empty")
    }
    if strings.ContainsAny(name, ".:") {
        return errors.New("session name cannot contain '.' or ':'")
    }
    for _, s := range existingSessions {
        if s.TmuxSession == name {
            return errors.New("session name already exists")
        }
    }
    return nil
}
```

## Key Bindings (Default)

| Action | Keys |
|--------|------|
| Move cursor right | `right`, `ctrl+f` |
| Move cursor left | `left`, `ctrl+b` |
| Delete backward | `backspace` |
| Delete word backward | `alt+backspace`, `ctrl+w` |
| Go to start | `home`, `ctrl+a` |
| Go to end | `end`, `ctrl+e` |
| Paste | `ctrl+v` |
