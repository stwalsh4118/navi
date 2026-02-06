package task

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProviderErrorType categorizes provider execution errors.
type ProviderErrorType int

const (
	ErrTimeout  ProviderErrorType = iota // Script exceeded timeout
	ErrParse                             // stdout was not valid JSON
	ErrExec                              // Non-zero exit code
	ErrNotFound                          // Script or built-in name not found
)

// envVarPrefix is the prefix for provider argument environment variables.
const envVarPrefix = "NAVI_TASK_ARG_"

// ProvidersDir is the directory where built-in provider scripts are located.
// It defaults to "providers" (relative to CWD) but can be overridden for testing
// or set to an absolute path at build time.
var ProvidersDir = "providers"

// builtinProviders maps shorthand names to script filenames within ProvidersDir.
var builtinProviders = map[string]string{
	"github-issues":  "github-issues.sh",
	"markdown-tasks": "markdown-tasks.sh",
}

// ProviderError is a structured error from provider execution.
type ProviderError struct {
	Type    ProviderErrorType
	Message string
	Stderr  string // captured stderr output
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Stderr)
	}
	return e.Message
}

// ResolveProvider resolves a provider name to an executable path.
// Built-in names ("github-issues", "markdown-tasks") resolve to bundled script paths under ProvidersDir.
// Relative paths are resolved relative to the project directory.
// Absolute paths are used as-is.
// Returns the resolved path and a ProviderError with ErrNotFound if not found.
func ResolveProvider(name string, projectDir string) (string, error) {
	var scriptPath string

	if filename, ok := builtinProviders[name]; ok {
		// Built-in provider: resolve relative to ProvidersDir.
		scriptPath = filepath.Join(ProvidersDir, filename)
	} else if filepath.IsAbs(name) {
		// Absolute path: use as-is.
		scriptPath = name
	} else {
		// Relative path: resolve relative to projectDir.
		scriptPath = filepath.Join(projectDir, name)
	}

	// Convert to absolute path so it works regardless of cmd.Dir in ExecuteProvider.
	scriptPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return "", &ProviderError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("failed to resolve provider path: %v", err),
		}
	}

	if _, err := os.Stat(scriptPath); err != nil {
		return "", &ProviderError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("provider script not found: %s", scriptPath),
		}
	}

	return scriptPath, nil
}

// ExecuteProvider runs a provider script and returns parsed results.
// It passes config args as NAVI_TASK_ARG_<KEY> environment variables (keys uppercased).
// Stdout is parsed as standard task JSON. Stderr is captured for error display.
// The script is killed if it exceeds the timeout.
func ExecuteProvider(config ProjectConfig, timeout time.Duration) (*ProviderResult, error) {
	scriptPath, err := ResolveProvider(config.Tasks.Provider, config.ProjectDir)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = config.ProjectDir
	cmd.Env = append(os.Environ(), buildEnvVars(config.Tasks.Args)...)
	// WaitDelay ensures child processes are killed after context cancellation.
	cmd.WaitDelay = time.Second

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &ProviderError{
				Type:    ErrTimeout,
				Message: fmt.Sprintf("provider timed out after %s", timeout),
				Stderr:  stderr.String(),
			}
		}
		return nil, &ProviderError{
			Type:    ErrExec,
			Message: fmt.Sprintf("provider exited with error: %v", err),
			Stderr:  stderr.String(),
		}
	}

	result, err := ParseProviderOutput(stdout.Bytes())
	if err != nil {
		return nil, &ProviderError{
			Type:    ErrParse,
			Message: fmt.Sprintf("failed to parse provider output: %v", err),
			Stderr:  stderr.String(),
		}
	}

	return result, nil
}

// buildEnvVars converts config args to NAVI_TASK_ARG_ environment variables.
func buildEnvVars(args map[string]string) []string {
	if len(args) == 0 {
		return nil
	}

	vars := make([]string, 0, len(args))
	for k, v := range args {
		vars = append(vars, envVarPrefix+strings.ToUpper(k)+"="+v)
	}
	return vars
}
