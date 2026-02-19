package tui

// DialogMode represents the type of dialog currently open.
type DialogMode int

// Dialog mode constants
const (
	DialogNone          DialogMode = iota // No dialog open
	DialogNewSession                      // New session creation dialog
	DialogKillConfirm                     // Kill session confirmation dialog
	DialogRename                          // Rename session dialog
	DialogGitDetail                       // Git detail view dialog
	DialogGitDiff                         // Git diff view dialog
	DialogMetricsDetail                   // Metrics detail view dialog
	DialogContentViewer                   // Content viewer overlay
	DialogSoundPackPicker                 // Sound pack picker overlay
)

// DialogTitle returns the title for a given dialog mode.
func DialogTitle(mode DialogMode) string {
	switch mode {
	case DialogNewSession:
		return "New Session"
	case DialogKillConfirm:
		return "Kill Session"
	case DialogRename:
		return "Rename Session"
	case DialogGitDetail:
		return "Git Information"
	case DialogGitDiff:
		return "Git Changes"
	case DialogMetricsDetail:
		return "Session Metrics"
	case DialogContentViewer:
		return "Content Viewer"
	case DialogSoundPackPicker:
		return "Sound Packs"
	default:
		return ""
	}
}
