package pm

import (
	"path/filepath"
	"testing"

	"github.com/stwalsh4118/navi/internal/git"
	"github.com/stwalsh4118/navi/internal/session"
	"github.com/stwalsh4118/navi/internal/task"
)

func TestResolveCurrentPBI_AcceptanceCriteriaScenarios(t *testing.T) {
	projectDir := t.TempDir()

	tests := []struct {
		name   string
		input  ResolverInput
		wantID string
		want   string
	}{
		{
			name: "provider hint current_pbi_id wins",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "feature/pbi-54-desc",
				TaskResult: &task.ProviderResult{
					CurrentPBIID:    "PBI-99",
					CurrentPBITitle: "Explicit Provider Hint",
					Groups: []task.TaskGroup{
						{ID: "PBI-54", Title: "Resolver", Status: "InProgress"},
					},
				},
			},
			wantID: "PBI-99",
			want:   "provider_hint",
		},
		{
			name: "provider hint is_current wins",
			input: ResolverInput{
				ProjectDir: projectDir,
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{
					{ID: "PBI-1", Title: "Done", Status: "Done"},
					{ID: "PBI-2", Title: "Current", Status: "Done", IsCurrent: true},
				}},
			},
			wantID: "PBI-2",
			want:   "provider_hint",
		},
		{
			name: "session metadata wins",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "feature/pbi-54-desc",
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{{ID: "PBI-54", Title: "Resolver", Status: "InProgress"}}},
				Sessions: []session.Info{
					{CWD: projectDir, CurrentPBI: "PBI-45", CurrentPBITitle: "Older", Timestamp: 10},
					{CWD: projectDir, CurrentPBI: "PBI-46", CurrentPBITitle: "Freshest", Timestamp: 20},
				},
			},
			wantID: "PBI-46",
			want:   "session_metadata",
		},
		{
			name: "branch pattern wins",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "feature/pbi-54-desc",
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{{ID: "PBI-54", Title: "Resolver", Status: "Done"}}},
			},
			wantID: "PBI-54",
			want:   "branch_pattern",
		},
		{
			name: "status heuristic wins when no explicit signals",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "main",
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{
					{ID: "PBI-10", Title: "Done", Status: "Done"},
					{ID: "PBI-11", Title: "Active", Status: "InProgress"},
				}},
			},
			wantID: "PBI-11",
			want:   "status_heuristic",
		},
		{
			name: "status heuristic ties are stable first in order",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "main",
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{
					{ID: "PBI-30", Title: "First InProgress", Status: "InProgress"},
					{ID: "PBI-31", Title: "Second InProgress", Status: "InProgress"},
				}},
			},
			wantID: "PBI-30",
			want:   "status_heuristic",
		},
		{
			name: "first group fallback when statuses unrecognized",
			input: ResolverInput{
				ProjectDir: projectDir,
				Branch:     "main",
				TaskResult: &task.ProviderResult{Groups: []task.TaskGroup{
					{ID: "PBI-20", Title: "Unknown", Status: "Blocked"},
					{ID: "PBI-21", Title: "Queued", Status: "Queued"},
				}},
			},
			wantID: "PBI-20",
			want:   "first_group_fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCurrentPBI(tt.input)
			if got.PBIID != tt.wantID || got.Source != tt.want {
				t.Fatalf("ResolveCurrentPBI() = %+v, want id=%q source=%q", got, tt.wantID, tt.want)
			}
		})
	}
}

func TestResolveCurrentPBI_CustomBranchPatternsAndFallback(t *testing.T) {
	projectDir := t.TempDir()
	taskResult := &task.ProviderResult{Groups: []task.TaskGroup{{ID: "PBI-54", Title: "Resolver", Status: "InProgress"}}}

	matched := ResolveCurrentPBI(ResolverInput{
		TaskResult:     taskResult,
		ProjectDir:     projectDir,
		Branch:         "workitem-54-resolver",
		BranchPatterns: []string{`workitem-(\d+)`},
	})
	if matched.PBIID != "PBI-54" || matched.Source != "branch_pattern" {
		t.Fatalf("custom branch pattern match failed: %+v", matched)
	}

	nonMatched := ResolveCurrentPBI(ResolverInput{
		TaskResult:     taskResult,
		ProjectDir:     projectDir,
		Branch:         "main",
		BranchPatterns: []string{`workitem-(\d+)`},
	})
	if nonMatched.PBIID != "PBI-54" || nonMatched.Source != "status_heuristic" {
		t.Fatalf("custom branch non-match fallback failed: %+v", nonMatched)
	}
}

func TestResolveCurrentPBI_MultipleSessionsConcurrentNoCollision(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, "navi")
	otherProjectDir := filepath.Join(home, "other")

	sessions := []session.Info{
		{CWD: "~/navi", CurrentPBI: "PBI-54", CurrentPBITitle: "Older", Timestamp: 10},
		{CWD: projectDir, CurrentPBI: "PBI-55", CurrentPBITitle: "Freshest", Timestamp: 20},
		{CWD: otherProjectDir, CurrentPBI: "PBI-1", CurrentPBITitle: "Other Project", Timestamp: 30},
	}

	result := ResolveCurrentPBI(ResolverInput{Sessions: sessions, ProjectDir: projectDir})
	if result.PBIID != "PBI-55" || result.Title != "Freshest" || result.Source != "session_metadata" {
		t.Fatalf("concurrent session metadata failed: %+v", result)
	}
}

func TestCaptureSnapshot_CurrentPBISourceIsDocumentedValue(t *testing.T) {
	projectDir := t.TempDir()

	originalGitInfo := gitInfoFunc
	originalHeadSHA := getHeadSHAFunc
	t.Cleanup(func() {
		gitInfoFunc = originalGitInfo
		getHeadSHAFunc = originalHeadSHA
	})

	gitInfoFunc = func(_ string) *git.Info {
		return &git.Info{Branch: "main", Ahead: 0, Dirty: false, PRNum: 0}
	}
	getHeadSHAFunc = func(_ string) string { return "sha" }

	snapshot := CaptureSnapshot(projectDir, nil, &task.ProviderResult{
		CurrentPBIID:    "PBI-54",
		CurrentPBITitle: "Resolver",
		Groups: []task.TaskGroup{
			{ID: "PBI-54", Title: "Resolver", Status: "InProgress"},
		},
	})

	if snapshot.CurrentPBISource == "" {
		t.Fatal("expected current_pbi_source to be non-empty")
	}
	allowed := map[string]bool{
		"provider_hint":        true,
		"session_metadata":     true,
		"branch_pattern":       true,
		"status_heuristic":     true,
		"first_group_fallback": true,
	}
	if !allowed[snapshot.CurrentPBISource] {
		t.Fatalf("current_pbi_source = %q, expected documented value", snapshot.CurrentPBISource)
	}
}
