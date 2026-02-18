package pm

import "testing"

func TestInferPBIFromBranch_DefaultPatterns(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		wantID string
		wantOK bool
	}{
		{name: "feature pattern", branch: "feature/pbi-54-session-scoped-pbi-resolution", wantID: "54", wantOK: true},
		{name: "pbi prefix", branch: "pbi-12", wantID: "12", wantOK: true},
		{name: "pbi prefix with slash", branch: "pbi-99/task", wantID: "99", wantOK: true},
		{name: "numeric prefix", branch: "54-session-scoped", wantID: "54", wantOK: true},
		{name: "main no match", branch: "main", wantID: "", wantOK: false},
		{name: "develop no match", branch: "develop", wantID: "", wantOK: false},
		{name: "hotfix no match", branch: "hotfix/auth-bug", wantID: "", wantOK: false},
		{name: "release version no match", branch: "release/v2-prepare", wantID: "", wantOK: false},
		{name: "issue suffix no match", branch: "hotfix/issue-123-fix", wantID: "", wantOK: false},
		{name: "empty branch", branch: "", wantID: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := InferPBIFromBranch(tt.branch, nil)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Fatalf("InferPBIFromBranch(%q, nil) = (%q, %v), want (%q, %v)", tt.branch, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestInferPBIFromBranch_CustomPatterns(t *testing.T) {
	patterns := []string{`ticket-(\d+)`, `task-(\d+)`}

	gotID, gotOK := InferPBIFromBranch("feature/task-88-refactor", patterns)
	if !gotOK || gotID != "88" {
		t.Fatalf("InferPBIFromBranch custom match = (%q, %v), want (%q, true)", gotID, gotOK, "88")
	}

	gotID, gotOK = InferPBIFromBranch("feature/pbi-54-session", patterns)
	if gotOK || gotID != "" {
		t.Fatalf("InferPBIFromBranch should not use defaults when custom patterns provided, got (%q, %v)", gotID, gotOK)
	}
}

func TestInferPBIFromBranch_InvalidPatternsAreIgnored(t *testing.T) {
	patterns := []string{`([`, `pbi-(\d+)`, `no-capture-group`}

	gotID, gotOK := InferPBIFromBranch("pbi-42-test", patterns)
	if !gotOK || gotID != "42" {
		t.Fatalf("InferPBIFromBranch with invalid patterns = (%q, %v), want (%q, true)", gotID, gotOK, "42")
	}
}

func TestInferPBIFromBranch_PatternPriority(t *testing.T) {
	patterns := []string{`feature/pbi-(\d+)`, `pbi-(\d+)`}

	gotID, gotOK := InferPBIFromBranch("feature/pbi-77-pbi-55", patterns)
	if !gotOK || gotID != "77" {
		t.Fatalf("InferPBIFromBranch priority = (%q, %v), want (%q, true)", gotID, gotOK, "77")
	}
}
