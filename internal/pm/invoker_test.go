package pm

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewInvokerInitializesStorage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}
	if invoker == nil {
		t.Fatal("expected invoker")
	}

	if _, err := os.Stat(resolveStoragePath(systemPromptFile)); err != nil {
		t.Fatalf("system prompt file should exist: %v", err)
	}
	if _, err := os.Stat(resolveStoragePath(outputSchemaFile)); err != nil {
		t.Fatalf("output schema file should exist: %v", err)
	}
}

func TestInvokeFreshSessionSuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	argsPath := t.TempDir() + "/args.txt"
	stdinPath := t.TempDir() + "/stdin.json"
	t.Setenv("INVOKER_HELPER_MODE", "fresh-success")
	t.Setenv("INVOKER_HELPER_ARGS_PATH", argsPath)
	t.Setenv("INVOKER_HELPER_STDIN_PATH", stdinPath)

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	result, err := invoker.Invoke(inbox)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if result.Output == nil || result.Output.Summary != "fresh-run" {
		t.Fatalf("unexpected output: %#v", result.Output)
	}

	argsContent, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("failed reading args fixture: %v", err)
	}
	argsText := string(argsContent)
	if strings.Contains(argsText, "--resume") {
		t.Fatalf("did not expect --resume in args: %s", argsText)
	}
	for _, flag := range []string{"--output-format", "--json-schema", "--tools", "--add-dir", "--system-prompt-file", "--model"} {
		if !strings.Contains(argsText, flag) {
			t.Fatalf("expected %s in args: %s", flag, argsText)
		}
	}

	stdinContent, err := os.ReadFile(stdinPath)
	if err != nil {
		t.Fatalf("failed reading stdin fixture: %v", err)
	}
	if !strings.Contains(string(stdinContent), `"trigger_type":"on_demand"`) {
		t.Fatalf("stdin should contain inbox JSON trigger_type, got: %s", string(stdinContent))
	}
}

func TestInvokeNeverResumes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	argsPath := t.TempDir() + "/args.txt"
	t.Setenv("INVOKER_HELPER_MODE", "fresh-success")
	t.Setenv("INVOKER_HELPER_ARGS_PATH", argsPath)

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerCommit, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	// Even after a successful invoke, the next invoke should NOT use --resume.
	for i := 0; i < 2; i++ {
		_, err := invoker.Invoke(inbox)
		if err != nil {
			t.Fatalf("Invoke %d failed: %v", i, err)
		}

		argsContent, err := os.ReadFile(argsPath)
		if err != nil {
			t.Fatalf("failed reading args fixture on run %d: %v", i, err)
		}
		if strings.Contains(string(argsContent), "--resume") {
			t.Fatalf("run %d: did not expect --resume in args: %s", i, string(argsContent))
		}
	}
}

func TestInvokeFailureReturnsInvokeError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "exit-failure")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	_, err = invoker.Invoke(inbox)
	if err == nil {
		t.Fatal("expected invoke error")
	}

	var invokeErr *InvokeError
	if !errors.As(err, &invokeErr) {
		t.Fatalf("expected InvokeError, got %T", err)
	}
	if invokeErr.ExitCode != 17 {
		t.Fatalf("exit code = %d, want 17", invokeErr.ExitCode)
	}
	if !strings.Contains(invokeErr.Stderr, "simulated failure") {
		t.Fatalf("stderr = %q, want simulated failure", invokeErr.Stderr)
	}
}

func TestInvokeTimeout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	restore := swapCommandContext(t)
	defer restore()

	t.Setenv("INVOKER_HELPER_MODE", "sleep")

	invoker, err := NewInvoker()
	if err != nil {
		t.Fatalf("NewInvoker failed: %v", err)
	}
	invoker.timeout = 10 * time.Millisecond

	inbox, err := BuildInbox(TriggerOnDemand, nil, nil)
	if err != nil {
		t.Fatalf("BuildInbox failed: %v", err)
	}

	_, err = invoker.Invoke(inbox)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var invokeErr *InvokeError
	if !errors.As(err, &invokeErr) {
		t.Fatalf("expected InvokeError, got %T", err)
	}
	if invokeErr.ExitCode != -1 {
		t.Fatalf("expected timeout/non-exit failure code -1, got %d", invokeErr.ExitCode)
	}
	if !errors.Is(invokeErr.Err, context.DeadlineExceeded) && !strings.Contains(invokeErr.Err.Error(), "killed") {
		t.Fatalf("expected timeout or killed process error, got %v", invokeErr.Err)
	}
}

func swapCommandContext(t *testing.T) func() {
	t.Helper()
	original := commandContext
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		helperArgs := []string{"-test.run=TestInvokerHelperProcess", "--"}
		helperArgs = append(helperArgs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}

	return func() {
		commandContext = original
	}
}

func TestInvokerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	marker := -1
	for i := range args {
		if args[i] == "--" {
			marker = i
			break
		}
	}
	if marker == -1 {
		os.Exit(2)
	}
	invokerArgs := args[marker+1:]

	if argsPath := os.Getenv("INVOKER_HELPER_ARGS_PATH"); argsPath != "" {
		_ = os.WriteFile(argsPath, []byte(strings.Join(invokerArgs, " ")), 0644)
	}
	if stdinPath := os.Getenv("INVOKER_HELPER_STDIN_PATH"); stdinPath != "" {
		stdinBytes, _ := os.ReadFile("/dev/stdin")
		_ = os.WriteFile(stdinPath, stdinBytes, 0644)
	}

	callCount := incrementHelperCounter(os.Getenv("INVOKER_HELPER_COUNTER_PATH"))

	switch os.Getenv("INVOKER_HELPER_MODE") {
	case "fresh-success":
		_, _ = os.Stdout.WriteString(`{"structured_output":{"summary":"fresh-run","projects":[],"attention_items":[],"breadcrumbs":[]}}`)
		os.Exit(0)
	case "exit-failure":
		_, _ = os.Stderr.WriteString("simulated failure")
		os.Exit(17)
	case "parse-failure":
		_, _ = os.Stdout.WriteString("not-json")
		os.Exit(0)
	case "rate-limit-then-success":
		if callCount <= 2 {
			_, _ = os.Stderr.WriteString("rate limit exceeded")
			os.Exit(1)
		}
		_, _ = os.Stdout.WriteString(`{"structured_output":{"summary":"rate-recovered","projects":[],"attention_items":[],"breadcrumbs":[]}}`)
		os.Exit(0)
	case "always-rate-limit":
		_, _ = os.Stderr.WriteString("rate limit exceeded")
		os.Exit(1)
	case "sleep":
		time.Sleep(200 * time.Millisecond)
		_, _ = os.Stdout.WriteString(`{"structured_output":{"summary":"late","projects":[],"attention_items":[],"breadcrumbs":[]}}`)
		os.Exit(0)
	default:
		_, _ = os.Stdout.WriteString(`{"structured_output":{"summary":"default","projects":[],"attention_items":[],"breadcrumbs":[]}}`)
		os.Exit(0)
	}
}

func incrementHelperCounter(path string) int {
	if path == "" {
		return 0
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		_ = os.WriteFile(path, []byte("1"), 0644)
		return 1
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		count = 0
	}
	count++
	_ = os.WriteFile(path, []byte(strconv.Itoa(count)), 0644)
	return count
}
