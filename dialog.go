package main

// DialogMode represents the type of dialog currently open.
type DialogMode int

// Dialog mode constants
const (
	DialogNone        DialogMode = iota // No dialog open
	DialogNewSession                    // New session creation dialog
	DialogKillConfirm                   // Kill session confirmation dialog
	DialogRename                        // Rename session dialog
)

// dialogTitle returns the title for a given dialog mode.
func dialogTitle(mode DialogMode) string {
	switch mode {
	case DialogNewSession:
		return "New Session"
	case DialogKillConfirm:
		return "Kill Session"
	case DialogRename:
		return "Rename Session"
	default:
		return ""
	}
}
