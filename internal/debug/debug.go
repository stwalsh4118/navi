package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Debug logging - enabled by setting NAVI_DEBUG=1
var (
	debugEnabled bool
	debugFile    *os.File
	debugMu      sync.Mutex
)

func init() {
	if os.Getenv("NAVI_DEBUG") == "1" {
		debugEnabled = true
		initDebugLog()
	}
}

func initDebugLog() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	logDir := filepath.Join(home, ".config", "navi")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	logPath := filepath.Join(logDir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}

	debugFile = f
	Log("=== navi started ===")
}

// Log writes a message to the debug log if debugging is enabled.
func Log(format string, args ...interface{}) {
	if !debugEnabled || debugFile == nil {
		return
	}

	debugMu.Lock()
	defer debugMu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(debugFile, "[%s] %s\n", timestamp, msg)
	debugFile.Sync()
}
