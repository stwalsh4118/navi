package pm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/stwalsh4118/navi/internal/debug"
)

const defaultInvokeTimeout = 120 * time.Second

var commandContext = exec.CommandContext

type Invoker struct {
	storageDir       string
	systemPromptPath string
	outputSchemaPath string
	commandName      string
	timeout          time.Duration
}

type InvokeResult struct {
	Output *PMBriefing
	Raw    []byte
	Usage  *InvokeUsage
}

// InvokeUsage holds token and cost stats from a Claude CLI invocation.
type InvokeUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	DurationMS   int     `json:"duration_ms"`
	NumTurns     int     `json:"num_turns"`
}

type InvokeError struct {
	ExitCode int
	Stderr   string
	Err      error
}

func (e *InvokeError) Error() string {
	if e.Stderr == "" {
		return fmt.Sprintf("pm invoke failed (exit %d): %v", e.ExitCode, e.Err)
	}
	return fmt.Sprintf("pm invoke failed (exit %d): %s", e.ExitCode, e.Stderr)
}

func (e *InvokeError) Unwrap() error {
	return e.Err
}

// StreamEvent carries a status update from a streaming invocation.
type StreamEvent struct {
	Status string // human-readable status like "Thinking...", "Reading file.go..."
}

func NewInvoker() (*Invoker, error) {
	if err := EnsureStorageLayout(); err != nil {
		return nil, err
	}

	return &Invoker{
		storageDir:       resolveStoragePath(pmDir),
		systemPromptPath: resolveStoragePath(systemPromptFile),
		outputSchemaPath: resolveStoragePath(outputSchemaFile),
		commandName:      "claude",
		timeout:          defaultInvokeTimeout,
	}, nil
}

// Invoke runs claude in non-streaming json mode (used by tests and as fallback).
func (i *Invoker) Invoke(inbox *InboxPayload) (*InvokeResult, error) {
	return i.InvokeStream(inbox, nil)
}

// InvokeStream runs claude with stream-json output. Status updates are sent to
// the stream channel if non-nil; callers can pass nil for non-streaming behavior.
func (i *Invoker) InvokeStream(inbox *InboxPayload, stream chan<- StreamEvent) (*InvokeResult, error) {
	if i == nil {
		return nil, errors.New("invoker is nil")
	}
	if inbox == nil {
		return nil, errors.New("inbox is nil")
	}

	inboxJSON, err := InboxToJSON(inbox)
	if err != nil {
		return nil, err
	}

	schemaJSON, err := os.ReadFile(i.outputSchemaPath)
	if err != nil {
		return nil, err
	}

	// Each invocation is a fresh session. Memory files provide continuity
	// between runs — resuming would accumulate conversation history and
	// inflate token costs without benefit.
	useStreaming := stream != nil
	args := i.buildArgs(string(schemaJSON), useStreaming)
	debug.Log("pm: invoking %s (streaming=%t), inbox=%d bytes", i.commandName, useStreaming, len(inboxJSON))

	ctx, cancel := context.WithTimeout(context.Background(), i.timeout)
	defer cancel()

	cmd := commandContext(ctx, i.commandName, args...)
	cmd.Stdin = bytes.NewReader(inboxJSON)
	baseEnv := cmd.Env
	if baseEnv == nil {
		baseEnv = os.Environ()
	}
	cmd.Env = filterEnvKey(baseEnv, "CLAUDECODE")

	if useStreaming {
		return i.runStreaming(ctx, cmd, stream)
	}
	return i.runBuffered(cmd)
}

// runBuffered executes the command and reads all output after exit (json mode).
func (i *Invoker) runBuffered(cmd *exec.Cmd) (*InvokeResult, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)
	debug.Log("pm: buffered finished in %s, stdout=%d, stderr=%d", elapsed, stdout.Len(), stderr.Len())

	if err != nil {
		invokeErr := &InvokeError{ExitCode: extractExitCode(err), Stderr: stderr.String(), Err: err}
		debug.Log("pm: invoke error: exit=%d, stderr=%q", invokeErr.ExitCode, invokeErr.Stderr)
		return nil, invokeErr
	}

	raw := stdout.Bytes()
	if len(raw) == 0 {
		debug.Log("pm: stdout empty, reading from stderr")
		raw = stderr.Bytes()
	}
	return i.parseResult(raw)
}

// runStreaming executes the command with stream-json output, sending status
// updates through the stream channel and collecting the final result line.
func (i *Invoker) runStreaming(ctx context.Context, cmd *exec.Cmd, stream chan<- StreamEvent) (*InvokeResult, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, &InvokeError{ExitCode: -1, Err: err}
	}

	sendStatus := func(status string) {
		select {
		case stream <- StreamEvent{Status: status}:
		case <-ctx.Done():
		}
	}

	sendStatus("Connected")

	var resultLine []byte
	var candidateJSON []byte // last assistant text that looked like JSON
	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		status, isResult, textContent := parseStreamLine(line)
		debug.Log("pm: stream: %s (result=%t)", status, isResult)

		if isResult {
			resultLine = append([]byte(nil), line...)
		}
		// Capture assistant text blocks that look like JSON briefings.
		if textContent != "" && textContent[0] == '{' {
			candidateJSON = []byte(textContent)
			debug.Log("pm: captured candidate JSON (%d bytes)", len(candidateJSON))
		}
		sendStatus(status)
	}

	if err := scanner.Err(); err != nil {
		debug.Log("pm: scanner error: %v", err)
	}

	waitErr := cmd.Wait()
	elapsed := time.Since(start)
	debug.Log("pm: streaming finished in %s, stderr=%d bytes", elapsed, stderr.Len())

	if waitErr != nil {
		invokeErr := &InvokeError{ExitCode: extractExitCode(waitErr), Stderr: stderr.String(), Err: waitErr}
		debug.Log("pm: stream error: exit=%d, stderr=%q", invokeErr.ExitCode, invokeErr.Stderr)
		return nil, invokeErr
	}

	if len(resultLine) == 0 {
		// Fall back to stderr in case stream-json wasn't supported.
		raw := stderr.Bytes()
		if len(raw) == 0 {
			return nil, errors.New("no result in stream output")
		}
		return i.parseResult(raw)
	}

	result, err := i.parseResult(resultLine)
	if err == nil {
		return result, nil
	}

	if len(candidateJSON) == 0 {
		return nil, err
	}

	// The result envelope's text was not valid JSON (e.g. Claude appended
	// a markdown summary after the JSON). Try the captured JSON text instead.
	debug.Log("pm: result parse failed, trying captured candidate JSON (%d bytes)", len(candidateJSON))
	briefing, parseErr := ParseOutput(candidateJSON)
	if parseErr != nil {
		debug.Log("pm: candidate JSON also failed: %v", parseErr)
		return nil, err
	}

	debug.Log("pm: candidate JSON parsed successfully")

	return &InvokeResult{
		Output: briefing,
		Raw:    candidateJSON,
	}, nil
}

// parseStreamLine extracts a human-readable status from a stream-json line.
// Returns the status string, whether this is the final result line, and the
// full text content of any assistant text block (empty string otherwise).
func parseStreamLine(line []byte) (string, bool, string) {
	var envelope struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Message struct {
			Content []struct {
				Type  string `json:"type"`
				Text  string `json:"text"`
				Name  string `json:"name"`
				Input json.RawMessage `json:"input"`
			} `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return "Processing...", false, ""
	}

	switch envelope.Type {
	case "system":
		return "Connected", false, ""
	case "rate_limit_event":
		return "Rate limit OK", false, ""
	case "result":
		return "Complete", true, ""
	case "assistant":
		for _, block := range envelope.Message.Content {
			switch block.Type {
			case "tool_use":
				return formatToolStatus(block.Name, block.Input), false, ""
			case "text":
				if block.Text != "" {
					display := block.Text
					if len(display) > 80 {
						display = display[:80] + "..."
					}
					return display, false, block.Text
				}
			}
		}
		return "Thinking...", false, ""
	default:
		return "Processing...", false, ""
	}
}

// formatToolStatus turns a tool call into a readable status line.
func formatToolStatus(toolName string, input json.RawMessage) string {
	var params map[string]interface{}
	if err := json.Unmarshal(input, &params); err != nil {
		return "Using " + toolName + "..."
	}

	switch toolName {
	case "Read":
		if path, ok := params["file_path"].(string); ok {
			return "Reading " + shortenPath(path)
		}
	case "Edit":
		if path, ok := params["file_path"].(string); ok {
			return "Editing " + shortenPath(path)
		}
	case "Write":
		if path, ok := params["file_path"].(string); ok {
			return "Writing " + shortenPath(path)
		}
	}
	return "Using " + toolName + "..."
}

// shortenPath returns the last 2 path segments for display.
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}

func (i *Invoker) parseResult(raw []byte) (*InvokeResult, error) {
	briefing, err := ParseOutput(raw)
	if err != nil {
		debug.Log("pm: parse output failed: %v (raw=%d bytes)", err, len(raw))
		return nil, err
	}
	debug.Log("pm: parsed briefing: summary=%q, projects=%d, attention=%d",
		briefing.Summary, len(briefing.Projects), len(briefing.AttentionItems))

	usage := extractUsage(raw)
	if usage != nil {
		debug.Log("pm: usage: input=%d, output=%d, cost=$%.4f, duration=%dms, turns=%d",
			usage.InputTokens, usage.OutputTokens, usage.CostUSD, usage.DurationMS, usage.NumTurns)
	}

	return &InvokeResult{
		Output: briefing,
		Raw:    raw,
		Usage:  usage,
	}, nil
}

func (i *Invoker) buildArgs(schemaJSON string, streaming bool) []string {
	args := []string{"-p"}

	if streaming {
		// stream-json gives us live status updates. --json-schema only works
		// with --output-format json, so we rely on the system prompt for
		// JSON formatting and parse the result text ourselves.
		args = append(args,
			"--output-format", "stream-json",
			"--verbose",
		)
	} else {
		// Non-streaming: use json + json-schema for guaranteed structured output.
		args = append(args,
			"--output-format", "json",
			"--json-schema", schemaJSON,
		)
	}

	args = append(args,
		"--model", "sonnet",
		"--tools", "Read,Write",
		"--permission-mode", "acceptEdits",
		"--add-dir", i.storageDir,
		"--system-prompt-file", i.systemPromptPath,
	)
	return args
}

func extractUsage(raw []byte) *InvokeUsage {
	var envelope struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		TotalCostUSD float64 `json:"total_cost_usd"`
		DurationMS   int     `json:"duration_ms"`
		NumTurns     int     `json:"num_turns"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	if envelope.Usage.InputTokens == 0 && envelope.Usage.OutputTokens == 0 {
		return nil
	}
	return &InvokeUsage{
		InputTokens:  envelope.Usage.InputTokens,
		OutputTokens: envelope.Usage.OutputTokens,
		CostUSD:      envelope.TotalCostUSD,
		DurationMS:   envelope.DurationMS,
		NumTurns:     envelope.NumTurns,
	}
}

func extractExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// filterEnvKey returns a copy of env with the given key removed.
func filterEnvKey(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
