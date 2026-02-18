package pm

import (
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestSelectGroupByStatus(t *testing.T) {
	t.Run("single inprogress group", func(t *testing.T) {
		id, title, ok := selectGroupByStatus([]task.TaskGroup{{ID: "PBI-54", Title: "Resolver", Status: "InProgress"}})
		if !ok || id != "PBI-54" || title != "Resolver" {
			t.Fatalf("selectGroupByStatus = (%q, %q, %v), want (%q, %q, true)", id, title, ok, "PBI-54", "Resolver")
		}
	})

	t.Run("mixed statuses inprogress wins", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "PBI-1", Title: "Done", Status: "Done"},
			{ID: "PBI-2", Title: "Agreed", Status: "Agreed"},
			{ID: "PBI-3", Title: "Current", Status: "InProgress"},
			{ID: "PBI-4", Title: "Proposed", Status: "Proposed"},
		}
		id, _, ok := selectGroupByStatus(groups)
		if !ok || id != "PBI-3" {
			t.Fatalf("selectGroupByStatus mixed = (%q, %v), want (%q, true)", id, ok, "PBI-3")
		}
	})

	t.Run("agreed wins when no inprogress", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "PBI-1", Title: "Done", Status: "Done"},
			{ID: "PBI-2", Title: "Ready", Status: "Agreed"},
		}
		id, _, ok := selectGroupByStatus(groups)
		if !ok || id != "PBI-2" {
			t.Fatalf("selectGroupByStatus agreed fallback = (%q, %v), want (%q, true)", id, ok, "PBI-2")
		}
	})

	t.Run("all done returns first done", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "PBI-1", Title: "Done One", Status: "Done"},
			{ID: "PBI-2", Title: "Done Two", Status: "Done"},
		}
		id, title, ok := selectGroupByStatus(groups)
		if !ok || id != "PBI-1" || title != "Done One" {
			t.Fatalf("selectGroupByStatus all done = (%q, %q, %v), want (%q, %q, true)", id, title, ok, "PBI-1", "Done One")
		}
	})

	t.Run("empty groups returns false", func(t *testing.T) {
		id, title, ok := selectGroupByStatus(nil)
		if ok || id != "" || title != "" {
			t.Fatalf("selectGroupByStatus empty = (%q, %q, %v), want empty result and false", id, title, ok)
		}
	})

	t.Run("unknown statuses return false", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "PBI-1", Title: "Blocked work", Status: "Blocked"},
			{ID: "PBI-2", Title: "No status", Status: ""},
		}
		id, title, ok := selectGroupByStatus(groups)
		if ok || id != "" || title != "" {
			t.Fatalf("selectGroupByStatus unknown statuses = (%q, %q, %v), want empty result and false", id, title, ok)
		}
	})

	t.Run("case insensitive status matching", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "PBI-1", Title: "lower", Status: "inprogress"},
			{ID: "PBI-2", Title: "mixed", Status: "InProgress"},
			{ID: "PBI-3", Title: "upper", Status: "INPROGRESS"},
		}
		id, _, ok := selectGroupByStatus(groups)
		if !ok || id != "PBI-1" {
			t.Fatalf("selectGroupByStatus case-insensitive = (%q, %v), want (%q, true)", id, ok, "PBI-1")
		}
	})

	t.Run("skip groups with empty id and title", func(t *testing.T) {
		groups := []task.TaskGroup{
			{ID: "", Title: "", Status: "InProgress"},
			{ID: "PBI-2", Title: "Valid", Status: "Agreed"},
		}
		id, _, ok := selectGroupByStatus(groups)
		if !ok || id != "PBI-2" {
			t.Fatalf("selectGroupByStatus skipped empty group = (%q, %v), want (%q, true)", id, ok, "PBI-2")
		}
	})
}

func TestResolveCurrentPBI(t *testing.T) {
	projectDir := t.TempDir()

	baseTaskResult := &task.ProviderResult{
		Groups: []task.TaskGroup{
			{ID: "PBI-1", Title: "One", Status: "Done"},
			{ID: "PBI-54", Title: "Fifty Four", Status: "InProgress"},
		},
	}

	t.Run("strategy1 provider current_pbi_id wins", func(t *testing.T) {
		input := ResolverInput{
			TaskResult: &task.ProviderResult{
				CurrentPBIID:    "PBI-99",
				CurrentPBITitle: "Provider Choice",
				Groups:          baseTaskResult.Groups,
			},
			ProjectDir: projectDir,
			Branch:     "feature/pbi-54-desc",
		}

		result := ResolveCurrentPBI(input)
		if result.PBIID != "PBI-99" || result.Title != "Provider Choice" || result.Source != "provider_hint" {
			t.Fatalf("provider id strategy failed: %+v", result)
		}
	})

	t.Run("strategy1 provider is_current wins", func(t *testing.T) {
		input := ResolverInput{
			TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{
				{ID: "PBI-1", Title: "One", Status: "InProgress"},
				{ID: "PBI-2", Title: "Two", Status: "Done", IsCurrent: true},
			}},
			ProjectDir: projectDir,
			Branch:     "feature/pbi-54-desc",
		}

		result := ResolveCurrentPBI(input)
		if result.PBIID != "PBI-2" || result.Title != "Two" || result.Source != "provider_hint" {
			t.Fatalf("provider is_current strategy failed: %+v", result)
		}
	})

	t.Run("strategy2 session metadata freshest wins", func(t *testing.T) {
		sessions := []session.Info{
			{CWD: projectDir, CurrentPBI: "PBI-3", CurrentPBITitle: "Older", Timestamp: 10},
			{CWD: projectDir, CurrentPBI: "PBI-4", CurrentPBITitle: "Newer", Timestamp: 20},
		}

		result := ResolveCurrentPBI(ResolverInput{TaskResult: baseTaskResult, Sessions: sessions, ProjectDir: projectDir})
		if result.PBIID != "PBI-4" || result.Title != "Newer" || result.Source != "session_metadata" {
			t.Fatalf("session metadata strategy failed: %+v", result)
		}
	})

	t.Run("strategy3 branch pattern with group title", func(t *testing.T) {
		result := ResolveCurrentPBI(ResolverInput{TaskResult: baseTaskResult, ProjectDir: projectDir, Branch: "feature/pbi-54-desc"})
		if result.PBIID != "PBI-54" || result.Title != "Fifty Four" || result.Source != "branch_pattern" {
			t.Fatalf("branch strategy failed: %+v", result)
		}
	})

	t.Run("strategy3 branch pattern without group match", func(t *testing.T) {
		result := ResolveCurrentPBI(ResolverInput{TaskResult: baseTaskResult, ProjectDir: projectDir, Branch: "feature/pbi-77-desc"})
		if result.PBIID != "PBI-77" || result.Title != "" || result.Source != "branch_pattern" {
			t.Fatalf("branch no-match title strategy failed: %+v", result)
		}
	})

	t.Run("strategy4 status heuristic", func(t *testing.T) {
		taskResult := &task.ProviderResult{Groups: []task.TaskGroup{
			{ID: "PBI-9", Title: "Done", Status: "Done"},
			{ID: "PBI-10", Title: "Agreed", Status: "Agreed"},
		}}
		result := ResolveCurrentPBI(ResolverInput{TaskResult: taskResult, ProjectDir: projectDir, Branch: "main"})
		if result.PBIID != "PBI-10" || result.Source != "status_heuristic" {
			t.Fatalf("status heuristic strategy failed: %+v", result)
		}
	})

	t.Run("strategy5 first group fallback", func(t *testing.T) {
		taskResult := &task.ProviderResult{Groups: []task.TaskGroup{
			{ID: "PBI-20", Title: "Unknown", Status: "Blocked"},
			{ID: "PBI-21", Title: "Another", Status: "Queued"},
		}}
		result := ResolveCurrentPBI(ResolverInput{TaskResult: taskResult, ProjectDir: projectDir, Branch: "main"})
		if result.PBIID != "PBI-20" || result.Title != "Unknown" || result.Source != "first_group_fallback" {
			t.Fatalf("first group fallback strategy failed: %+v", result)
		}
	})

	t.Run("empty input returns zero value", func(t *testing.T) {
		result := ResolveCurrentPBI(ResolverInput{})
		if result != (ResolverResult{}) {
			t.Fatalf("expected zero-value result, got %+v", result)
		}
	})

	t.Run("precedence provider over session and branch", func(t *testing.T) {
		sessions := []session.Info{{CWD: projectDir, CurrentPBI: "PBI-1", Timestamp: 100}}
		result := ResolveCurrentPBI(ResolverInput{
			TaskResult: &task.ProviderResult{
				CurrentPBIID:    "PBI-88",
				CurrentPBITitle: "Provider Wins",
				Groups:          baseTaskResult.Groups,
			},
			Sessions:   sessions,
			ProjectDir: projectDir,
			Branch:     "feature/pbi-54-desc",
		})
		if result.PBIID != "PBI-88" || result.Source != "provider_hint" {
			t.Fatalf("precedence check failed: %+v", result)
		}
	})

	t.Run("session metadata matches expanded cwd", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		project := filepath.Join(home, "proj")
		sessions := []session.Info{{CWD: "~/proj", CurrentPBI: "PBI-31", Timestamp: 30}}

		result := ResolveCurrentPBI(ResolverInput{TaskResult: baseTaskResult, Sessions: sessions, ProjectDir: project})
		if result.PBIID != "PBI-31" || result.Source != "session_metadata" {
			t.Fatalf("expanded cwd matching failed: %+v", result)
		}
	})
}
