package git

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCheckSummary_IsAllPassed(t *testing.T) {
	tests := []struct {
		name string
		cs   CheckSummary
		want bool
	}{
		{"all passed", CheckSummary{Total: 3, Passed: 3}, true},
		{"some failed", CheckSummary{Total: 3, Passed: 2, Failed: 1}, false},
		{"some pending", CheckSummary{Total: 3, Passed: 2, Pending: 1}, false},
		{"no checks", CheckSummary{Total: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cs.IsAllPassed(); got != tt.want {
				t.Errorf("IsAllPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSummary_HasFailures(t *testing.T) {
	tests := []struct {
		name string
		cs   CheckSummary
		want bool
	}{
		{"no failures", CheckSummary{Total: 3, Passed: 3}, false},
		{"has failures", CheckSummary{Total: 3, Passed: 2, Failed: 1}, true},
		{"no checks", CheckSummary{Total: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cs.HasFailures(); got != tt.want {
				t.Errorf("HasFailures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSummary_IsPending(t *testing.T) {
	tests := []struct {
		name string
		cs   CheckSummary
		want bool
	}{
		{"no pending", CheckSummary{Total: 3, Passed: 3}, false},
		{"has pending", CheckSummary{Total: 3, Passed: 2, Pending: 1}, true},
		{"no checks", CheckSummary{Total: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cs.IsPending(); got != tt.want {
				t.Errorf("IsPending() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckSummary_CheckIndicator(t *testing.T) {
	tests := []struct {
		name string
		cs   CheckSummary
		want string
	}{
		{"all passed", CheckSummary{Total: 3, Passed: 3}, IndicatorPass},
		{"has failures", CheckSummary{Total: 3, Passed: 2, Failed: 1}, IndicatorFail},
		{"has pending", CheckSummary{Total: 3, Passed: 2, Pending: 1}, IndicatorPending},
		{"failure takes priority over pending", CheckSummary{Total: 3, Passed: 1, Failed: 1, Pending: 1}, IndicatorFail},
		{"no checks", CheckSummary{Total: 0}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cs.CheckIndicator(); got != tt.want {
				t.Errorf("CheckIndicator() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRDetail_IsStale(t *testing.T) {
	tests := []struct {
		name string
		pr   *PRDetail
		want bool
	}{
		{"nil PRDetail", nil, true},
		{"zero FetchedAt", &PRDetail{FetchedAt: 0}, true},
		{"recent fetch", &PRDetail{FetchedAt: time.Now().Unix()}, false},
		{"old fetch", &PRDetail{FetchedAt: time.Now().Add(-2 * PRCacheMaxAge).Unix()}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pr.IsStale(); got != tt.want {
				t.Errorf("IsStale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRDetail_MergeIndicator(t *testing.T) {
	tests := []struct {
		name      string
		mergeable string
		want      string
	}{
		{"mergeable", MergeableMergeable, IndicatorPass + " No conflicts"},
		{"conflicting", MergeableConflicting, IndicatorFail + " Has conflicts"},
		{"unknown", MergeableUnknown, IndicatorUnknown + " Unknown"},
		{"empty", "", IndicatorUnknown + " Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PRDetail{Mergeable: tt.mergeable}
			if got := pr.MergeIndicator(); got != tt.want {
				t.Errorf("MergeIndicator() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRDetail_ReviewIndicator(t *testing.T) {
	tests := []struct {
		name     string
		pr       *PRDetail
		contains string
	}{
		{
			"approved with reviewers",
			&PRDetail{
				ReviewStatus: ReviewApproved,
				Reviewers:    []Reviewer{{Login: "alice", State: ReviewerApproved}},
			},
			"Approved by alice",
		},
		{
			"changes requested",
			&PRDetail{
				ReviewStatus: ReviewChangesRequired,
				Reviewers:    []Reviewer{{Login: "bob", State: ReviewerChangesRequired}},
			},
			"Changes requested by bob",
		},
		{
			"review pending",
			&PRDetail{ReviewStatus: ReviewRequired},
			"Review pending",
		},
		{
			"no reviewers",
			&PRDetail{},
			"No reviewers assigned",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pr.ReviewIndicator()
			if got == "" {
				t.Fatal("ReviewIndicator() returned empty string")
			}
			// Just check it contains the expected text
			if !containsStr(got, tt.contains) {
				t.Errorf("ReviewIndicator() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPRDetailJSON(t *testing.T) {
	original := &PRDetail{
		Number:       42,
		Title:        "Add OAuth2 flow",
		State:        PRStateOpen,
		Draft:        false,
		Mergeable:    MergeableMergeable,
		Labels:       []string{"enhancement", "auth"},
		ChangedFiles: 12,
		Additions:    340,
		Deletions:    85,
		ReviewStatus: ReviewApproved,
		Reviewers: []Reviewer{
			{Login: "alice", State: ReviewerApproved},
			{Login: "bob", State: ReviewerApproved},
		},
		Comments: 5,
		Checks: []Check{
			{Name: "build", Status: CheckStatusCompleted, Conclusion: CheckConclusionSuccess},
			{Name: "lint", Status: CheckStatusCompleted, Conclusion: CheckConclusionSuccess},
			{Name: "test", Status: CheckStatusInProgress, Conclusion: ""},
		},
		CheckSummary: CheckSummary{Total: 3, Passed: 2, Pending: 1},
		URL:          "https://github.com/user/repo/pull/42",
		FetchedAt:    time.Now().Unix(),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal PRDetail: %v", err)
	}

	var decoded PRDetail
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PRDetail: %v", err)
	}

	if decoded.Number != original.Number {
		t.Errorf("Number mismatch: got %d, want %d", decoded.Number, original.Number)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title mismatch: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.State != original.State {
		t.Errorf("State mismatch: got %q, want %q", decoded.State, original.State)
	}
	if decoded.Draft != original.Draft {
		t.Errorf("Draft mismatch: got %v, want %v", decoded.Draft, original.Draft)
	}
	if decoded.Mergeable != original.Mergeable {
		t.Errorf("Mergeable mismatch: got %q, want %q", decoded.Mergeable, original.Mergeable)
	}
	if len(decoded.Labels) != len(original.Labels) {
		t.Errorf("Labels length mismatch: got %d, want %d", len(decoded.Labels), len(original.Labels))
	}
	if decoded.ChangedFiles != original.ChangedFiles {
		t.Errorf("ChangedFiles mismatch: got %d, want %d", decoded.ChangedFiles, original.ChangedFiles)
	}
	if decoded.Additions != original.Additions {
		t.Errorf("Additions mismatch: got %d, want %d", decoded.Additions, original.Additions)
	}
	if decoded.Deletions != original.Deletions {
		t.Errorf("Deletions mismatch: got %d, want %d", decoded.Deletions, original.Deletions)
	}
	if len(decoded.Reviewers) != len(original.Reviewers) {
		t.Errorf("Reviewers length mismatch: got %d, want %d", len(decoded.Reviewers), len(original.Reviewers))
	}
	if decoded.Comments != original.Comments {
		t.Errorf("Comments mismatch: got %d, want %d", decoded.Comments, original.Comments)
	}
	if len(decoded.Checks) != len(original.Checks) {
		t.Errorf("Checks length mismatch: got %d, want %d", len(decoded.Checks), len(original.Checks))
	}
	if decoded.CheckSummary.Total != original.CheckSummary.Total {
		t.Errorf("CheckSummary.Total mismatch: got %d, want %d", decoded.CheckSummary.Total, original.CheckSummary.Total)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL mismatch: got %q, want %q", decoded.URL, original.URL)
	}
}

func TestInfoWithPRDetail(t *testing.T) {
	// Test Info with nil PRDetail (backward compat)
	info := &Info{
		Branch:    "main",
		FetchedAt: time.Now().Unix(),
	}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal Info with nil PRDetail: %v", err)
	}
	var decoded Info
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Info with nil PRDetail: %v", err)
	}
	if decoded.PRDetail != nil {
		t.Error("Expected nil PRDetail after round-trip")
	}

	// Test Info with populated PRDetail
	info.PRDetail = &PRDetail{
		Number: 42,
		Title:  "Test PR",
		State:  PRStateOpen,
	}
	info.PRNum = 42
	data, err = json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal Info with PRDetail: %v", err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Info with PRDetail: %v", err)
	}
	if decoded.PRDetail == nil {
		t.Fatal("Expected non-nil PRDetail after round-trip")
	}
	if decoded.PRDetail.Number != 42 {
		t.Errorf("PRDetail.Number mismatch: got %d, want 42", decoded.PRDetail.Number)
	}
	if decoded.PRNum != 42 {
		t.Errorf("PRNum mismatch: got %d, want 42", decoded.PRNum)
	}
}

func TestJoinNames(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"empty", nil, ""},
		{"one name", []string{"alice"}, "alice"},
		{"two names", []string{"alice", "bob"}, "alice and bob"},
		{"three names", []string{"alice", "bob", "charlie"}, "alice, bob, and charlie"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinNames(tt.names); got != tt.want {
				t.Errorf("joinNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseReviewComments(t *testing.T) {
	t.Run("parses valid review comments", func(t *testing.T) {
		input := `[
			{
				"user": {"login": "alice"},
				"body": "Looks good here",
				"created_at": "2025-01-15T10:30:00Z",
				"updated_at": "2025-01-15T10:30:00Z",
				"path": "main.go",
				"line": 42,
				"original_line": 40
			},
			{
				"user": {"login": "bob"},
				"body": "Consider refactoring",
				"created_at": "2025-01-15T11:00:00Z",
				"updated_at": "2025-01-15T11:05:00Z",
				"path": "utils.go",
				"line": 0,
				"original_line": 15
			}
		]`

		comments, err := parseReviewComments([]byte(input))
		if err != nil {
			t.Fatalf("parseReviewComments() error = %v", err)
		}
		if len(comments) != 2 {
			t.Fatalf("expected 2 comments, got %d", len(comments))
		}

		// First comment
		if comments[0].Author != "alice" {
			t.Errorf("comment[0].Author = %q, want %q", comments[0].Author, "alice")
		}
		if comments[0].Body != "Looks good here" {
			t.Errorf("comment[0].Body = %q, want %q", comments[0].Body, "Looks good here")
		}
		if comments[0].CreatedAt != "2025-01-15T10:30:00Z" {
			t.Errorf("comment[0].CreatedAt = %q, want %q", comments[0].CreatedAt, "2025-01-15T10:30:00Z")
		}
		if comments[0].Type != CommentTypeReview {
			t.Errorf("comment[0].Type = %q, want %q", comments[0].Type, CommentTypeReview)
		}
		if comments[0].FilePath != "main.go" {
			t.Errorf("comment[0].FilePath = %q, want %q", comments[0].FilePath, "main.go")
		}
		if comments[0].Line != 42 {
			t.Errorf("comment[0].Line = %d, want %d", comments[0].Line, 42)
		}

		// Second comment: line is 0, should fall back to original_line
		if comments[1].Author != "bob" {
			t.Errorf("comment[1].Author = %q, want %q", comments[1].Author, "bob")
		}
		if comments[1].Line != 15 {
			t.Errorf("comment[1].Line = %d, want %d (should use original_line fallback)", comments[1].Line, 15)
		}
		if comments[1].FilePath != "utils.go" {
			t.Errorf("comment[1].FilePath = %q, want %q", comments[1].FilePath, "utils.go")
		}
	})

	t.Run("parses empty array", func(t *testing.T) {
		comments, err := parseReviewComments([]byte(`[]`))
		if err != nil {
			t.Fatalf("parseReviewComments() error = %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(comments))
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := parseReviewComments([]byte(`not json`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("handles missing optional fields", func(t *testing.T) {
		input := `[{
			"user": {"login": "carol"},
			"body": "LGTM",
			"created_at": "2025-01-15T12:00:00Z",
			"updated_at": "",
			"path": "",
			"line": 0,
			"original_line": 0
		}]`
		comments, err := parseReviewComments([]byte(input))
		if err != nil {
			t.Fatalf("parseReviewComments() error = %v", err)
		}
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment, got %d", len(comments))
		}
		if comments[0].Line != 0 {
			t.Errorf("comment.Line = %d, want 0", comments[0].Line)
		}
		if comments[0].FilePath != "" {
			t.Errorf("comment.FilePath = %q, want empty", comments[0].FilePath)
		}
	})
}

func TestParseGeneralComments(t *testing.T) {
	t.Run("parses valid general comments", func(t *testing.T) {
		input := `[
			{
				"user": {"login": "alice"},
				"body": "Great work on this PR!",
				"created_at": "2025-01-15T09:00:00Z",
				"updated_at": "2025-01-15T09:00:00Z"
			},
			{
				"user": {"login": "bob"},
				"body": "Can we add tests?",
				"created_at": "2025-01-15T09:30:00Z",
				"updated_at": "2025-01-15T09:35:00Z"
			}
		]`

		comments, err := parseGeneralComments([]byte(input))
		if err != nil {
			t.Fatalf("parseGeneralComments() error = %v", err)
		}
		if len(comments) != 2 {
			t.Fatalf("expected 2 comments, got %d", len(comments))
		}

		if comments[0].Author != "alice" {
			t.Errorf("comment[0].Author = %q, want %q", comments[0].Author, "alice")
		}
		if comments[0].Body != "Great work on this PR!" {
			t.Errorf("comment[0].Body = %q, want %q", comments[0].Body, "Great work on this PR!")
		}
		if comments[0].Type != CommentTypeGeneral {
			t.Errorf("comment[0].Type = %q, want %q", comments[0].Type, CommentTypeGeneral)
		}
		// General comments should not have file path or line
		if comments[0].FilePath != "" {
			t.Errorf("comment[0].FilePath = %q, want empty", comments[0].FilePath)
		}
		if comments[0].Line != 0 {
			t.Errorf("comment[0].Line = %d, want 0", comments[0].Line)
		}

		if comments[1].Author != "bob" {
			t.Errorf("comment[1].Author = %q, want %q", comments[1].Author, "bob")
		}
		if comments[1].UpdatedAt != "2025-01-15T09:35:00Z" {
			t.Errorf("comment[1].UpdatedAt = %q, want %q", comments[1].UpdatedAt, "2025-01-15T09:35:00Z")
		}
	})

	t.Run("parses empty array", func(t *testing.T) {
		comments, err := parseGeneralComments([]byte(`[]`))
		if err != nil {
			t.Fatalf("parseGeneralComments() error = %v", err)
		}
		if len(comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(comments))
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := parseGeneralComments([]byte(`{invalid}`))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestSortCommentsByCreatedAt(t *testing.T) {
	comments := []PRComment{
		{Author: "charlie", CreatedAt: "2025-01-15T12:00:00Z", Type: CommentTypeGeneral},
		{Author: "alice", CreatedAt: "2025-01-15T09:00:00Z", Type: CommentTypeReview},
		{Author: "bob", CreatedAt: "2025-01-15T10:30:00Z", Type: CommentTypeGeneral},
	}

	sortCommentsByCreatedAt(comments)

	expectedOrder := []string{"alice", "bob", "charlie"}
	for i, expected := range expectedOrder {
		if comments[i].Author != expected {
			t.Errorf("after sort, comment[%d].Author = %q, want %q", i, comments[i].Author, expected)
		}
	}
}

func TestSortCommentsByCreatedAt_EmptySlice(t *testing.T) {
	var comments []PRComment
	sortCommentsByCreatedAt(comments) // Should not panic
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestSortCommentsByCreatedAt_SingleElement(t *testing.T) {
	comments := []PRComment{
		{Author: "alice", CreatedAt: "2025-01-15T09:00:00Z"},
	}
	sortCommentsByCreatedAt(comments)
	if comments[0].Author != "alice" {
		t.Errorf("single element sort: got %q, want %q", comments[0].Author, "alice")
	}
}

func TestSortCommentsByCreatedAt_MixedTypes(t *testing.T) {
	// Test that review and general comments interleave correctly by timestamp
	comments := []PRComment{
		{Author: "review2", CreatedAt: "2025-01-15T11:00:00Z", Type: CommentTypeReview},
		{Author: "general1", CreatedAt: "2025-01-15T09:00:00Z", Type: CommentTypeGeneral},
		{Author: "review1", CreatedAt: "2025-01-15T10:00:00Z", Type: CommentTypeReview},
		{Author: "general2", CreatedAt: "2025-01-15T10:30:00Z", Type: CommentTypeGeneral},
	}

	sortCommentsByCreatedAt(comments)

	expectedOrder := []string{"general1", "review1", "general2", "review2"}
	for i, expected := range expectedOrder {
		if comments[i].Author != expected {
			t.Errorf("after sort, comment[%d].Author = %q, want %q", i, comments[i].Author, expected)
		}
	}
}

func TestPRCommentJSON(t *testing.T) {
	original := PRComment{
		Author:    "alice",
		Body:      "Needs refactoring",
		CreatedAt: "2025-01-15T10:30:00Z",
		UpdatedAt: "2025-01-15T10:35:00Z",
		Type:      CommentTypeReview,
		FilePath:  "internal/git/pr.go",
		Line:      42,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal PRComment: %v", err)
	}

	var decoded PRComment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PRComment: %v", err)
	}

	if decoded.Author != original.Author {
		t.Errorf("Author mismatch: got %q, want %q", decoded.Author, original.Author)
	}
	if decoded.Body != original.Body {
		t.Errorf("Body mismatch: got %q, want %q", decoded.Body, original.Body)
	}
	if decoded.CreatedAt != original.CreatedAt {
		t.Errorf("CreatedAt mismatch: got %q, want %q", decoded.CreatedAt, original.CreatedAt)
	}
	if decoded.UpdatedAt != original.UpdatedAt {
		t.Errorf("UpdatedAt mismatch: got %q, want %q", decoded.UpdatedAt, original.UpdatedAt)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.FilePath != original.FilePath {
		t.Errorf("FilePath mismatch: got %q, want %q", decoded.FilePath, original.FilePath)
	}
	if decoded.Line != original.Line {
		t.Errorf("Line mismatch: got %d, want %d", decoded.Line, original.Line)
	}
}

func TestPRCommentJSON_OmitEmpty(t *testing.T) {
	// General comment without optional fields
	comment := PRComment{
		Author:    "bob",
		Body:      "LGTM",
		CreatedAt: "2025-01-15T09:00:00Z",
		Type:      CommentTypeGeneral,
	}

	data, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("Failed to marshal PRComment: %v", err)
	}

	// Verify omitted fields are not in JSON
	jsonStr := string(data)
	if containsStr(jsonStr, "file_path") {
		t.Error("Expected file_path to be omitted from JSON")
	}
	if containsStr(jsonStr, "\"line\"") {
		t.Error("Expected line to be omitted from JSON")
	}
	if containsStr(jsonStr, "updated_at") {
		t.Error("Expected updated_at to be omitted from JSON")
	}
}

func TestGetPRCommentsByRepo_ValidationErrors(t *testing.T) {
	t.Run("empty owner", func(t *testing.T) {
		_, err := GetPRCommentsByRepo("", "repo", 1)
		if err == nil {
			t.Error("expected error for empty owner")
		}
	})

	t.Run("empty repo", func(t *testing.T) {
		_, err := GetPRCommentsByRepo("owner", "", 1)
		if err == nil {
			t.Error("expected error for empty repo")
		}
	})

	t.Run("zero PR number", func(t *testing.T) {
		_, err := GetPRCommentsByRepo("owner", "repo", 0)
		if err == nil {
			t.Error("expected error for zero PR number")
		}
	})

	t.Run("negative PR number", func(t *testing.T) {
		_, err := GetPRCommentsByRepo("owner", "repo", -1)
		if err == nil {
			t.Error("expected error for negative PR number")
		}
	})
}

func TestParsePRDetailJSON_FullPayload(t *testing.T) {
	// Simulates a realistic gh pr view --json output
	input := `{
		"number": 42,
		"title": "feat: add OAuth2 flow",
		"state": "OPEN",
		"isDraft": false,
		"mergeable": "MERGEABLE",
		"labels": [
			{"name": "enhancement"},
			{"name": "auth"}
		],
		"changedFiles": 12,
		"additions": 340,
		"deletions": 85,
		"reviewDecision": "APPROVED",
		"reviews": [
			{"author": {"login": "alice"}, "state": "APPROVED", "body": "LGTM!"},
			{"author": {"login": "bob"}, "state": "APPROVED", "body": ""}
		],
		"comments": [
			{"body": "comment 1"},
			{"body": "comment 2"},
			{"body": "comment 3"}
		],
		"statusCheckRollup": [
			{"__typename": "CheckRun", "name": "build", "status": "COMPLETED", "conclusion": "SUCCESS"},
			{"__typename": "CheckRun", "name": "lint", "status": "COMPLETED", "conclusion": "SUCCESS"},
			{"__typename": "CheckRun", "name": "test", "status": "IN_PROGRESS", "conclusion": ""},
			{"__typename": "StatusContext", "context": "ci/circleci", "state": "SUCCESS"}
		],
		"url": "https://github.com/user/repo/pull/42"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil for valid input")
	}

	// Basic fields
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Title != "feat: add OAuth2 flow" {
		t.Errorf("Title = %q, want %q", pr.Title, "feat: add OAuth2 flow")
	}
	if pr.State != PRStateOpen {
		t.Errorf("State = %q, want %q", pr.State, PRStateOpen)
	}
	if pr.Draft {
		t.Error("Draft = true, want false")
	}
	if pr.Mergeable != MergeableMergeable {
		t.Errorf("Mergeable = %q, want %q", pr.Mergeable, MergeableMergeable)
	}
	if pr.ChangedFiles != 12 {
		t.Errorf("ChangedFiles = %d, want 12", pr.ChangedFiles)
	}
	if pr.Additions != 340 {
		t.Errorf("Additions = %d, want 340", pr.Additions)
	}
	if pr.Deletions != 85 {
		t.Errorf("Deletions = %d, want 85", pr.Deletions)
	}
	if pr.ReviewStatus != ReviewApproved {
		t.Errorf("ReviewStatus = %q, want %q", pr.ReviewStatus, ReviewApproved)
	}
	if pr.URL != "https://github.com/user/repo/pull/42" {
		t.Errorf("URL = %q, want %q", pr.URL, "https://github.com/user/repo/pull/42")
	}
	if pr.FetchedAt == 0 {
		t.Error("FetchedAt should be set")
	}

	// Labels
	if len(pr.Labels) != 2 {
		t.Fatalf("Labels length = %d, want 2", len(pr.Labels))
	}
	if pr.Labels[0] != "enhancement" {
		t.Errorf("Labels[0] = %q, want %q", pr.Labels[0], "enhancement")
	}
	if pr.Labels[1] != "auth" {
		t.Errorf("Labels[1] = %q, want %q", pr.Labels[1], "auth")
	}

	// Reviewers
	if len(pr.Reviewers) != 2 {
		t.Fatalf("Reviewers length = %d, want 2", len(pr.Reviewers))
	}
	if pr.Reviewers[0].Login != "alice" {
		t.Errorf("Reviewers[0].Login = %q, want %q", pr.Reviewers[0].Login, "alice")
	}
	if pr.Reviewers[0].State != "APPROVED" {
		t.Errorf("Reviewers[0].State = %q, want %q", pr.Reviewers[0].State, "APPROVED")
	}
	if pr.Reviewers[1].Login != "bob" {
		t.Errorf("Reviewers[1].Login = %q, want %q", pr.Reviewers[1].Login, "bob")
	}

	// Comments: 3 general comments + 1 review with body = 4
	if pr.Comments != 4 {
		t.Errorf("Comments = %d, want 4 (3 general + 1 review with body)", pr.Comments)
	}

	// Checks: 3 CheckRuns + 1 StatusContext = 4
	if len(pr.Checks) != 4 {
		t.Fatalf("Checks length = %d, want 4", len(pr.Checks))
	}

	// Check "build" - CheckRun, completed success
	if pr.Checks[0].Name != "build" {
		t.Errorf("Checks[0].Name = %q, want %q", pr.Checks[0].Name, "build")
	}
	if pr.Checks[0].Status != CheckStatusCompleted {
		t.Errorf("Checks[0].Status = %q, want %q", pr.Checks[0].Status, CheckStatusCompleted)
	}
	if pr.Checks[0].Conclusion != CheckConclusionSuccess {
		t.Errorf("Checks[0].Conclusion = %q, want %q", pr.Checks[0].Conclusion, CheckConclusionSuccess)
	}

	// Check "test" - CheckRun, in progress
	if pr.Checks[2].Name != "test" {
		t.Errorf("Checks[2].Name = %q, want %q", pr.Checks[2].Name, "test")
	}
	if pr.Checks[2].Status != CheckStatusInProgress {
		t.Errorf("Checks[2].Status = %q, want %q", pr.Checks[2].Status, CheckStatusInProgress)
	}
	if pr.Checks[2].Conclusion != "" {
		t.Errorf("Checks[2].Conclusion = %q, want empty", pr.Checks[2].Conclusion)
	}

	// Check "ci/circleci" - StatusContext, success
	if pr.Checks[3].Name != "ci/circleci" {
		t.Errorf("Checks[3].Name = %q, want %q", pr.Checks[3].Name, "ci/circleci")
	}
	if pr.Checks[3].Status != CheckStatusCompleted {
		t.Errorf("Checks[3].Status = %q, want %q (StatusContext maps to COMPLETED)", pr.Checks[3].Status, CheckStatusCompleted)
	}
	if pr.Checks[3].Conclusion != CheckConclusionSuccess {
		t.Errorf("Checks[3].Conclusion = %q, want %q", pr.Checks[3].Conclusion, CheckConclusionSuccess)
	}

	// CheckSummary: 2 passed (build, circleci), 1 in_progress (test), 1 passed (lint)
	// Total 4: build=passed, lint=passed, test=pending, circleci=passed -> 3 passed, 1 pending
	if pr.CheckSummary.Total != 4 {
		t.Errorf("CheckSummary.Total = %d, want 4", pr.CheckSummary.Total)
	}
	if pr.CheckSummary.Passed != 3 {
		t.Errorf("CheckSummary.Passed = %d, want 3", pr.CheckSummary.Passed)
	}
	if pr.CheckSummary.Failed != 0 {
		t.Errorf("CheckSummary.Failed = %d, want 0", pr.CheckSummary.Failed)
	}
	if pr.CheckSummary.Pending != 1 {
		t.Errorf("CheckSummary.Pending = %d, want 1", pr.CheckSummary.Pending)
	}
}

func TestParsePRDetailJSON_DraftPR(t *testing.T) {
	input := `{
		"number": 10,
		"title": "WIP: initial draft",
		"state": "OPEN",
		"isDraft": true,
		"mergeable": "UNKNOWN",
		"labels": [],
		"changedFiles": 1,
		"additions": 5,
		"deletions": 0,
		"reviewDecision": "",
		"reviews": [],
		"comments": [],
		"statusCheckRollup": [],
		"url": "https://github.com/user/repo/pull/10"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil for valid draft PR")
	}

	if !pr.Draft {
		t.Error("Draft = false, want true")
	}
	if pr.Mergeable != MergeableUnknown {
		t.Errorf("Mergeable = %q, want %q", pr.Mergeable, MergeableUnknown)
	}
	if len(pr.Labels) != 0 {
		t.Errorf("Labels length = %d, want 0", len(pr.Labels))
	}
	if len(pr.Reviewers) != 0 {
		t.Errorf("Reviewers length = %d, want 0", len(pr.Reviewers))
	}
	if pr.Comments != 0 {
		t.Errorf("Comments = %d, want 0", pr.Comments)
	}
	if len(pr.Checks) != 0 {
		t.Errorf("Checks length = %d, want 0", len(pr.Checks))
	}
	if pr.CheckSummary.Total != 0 {
		t.Errorf("CheckSummary.Total = %d, want 0", pr.CheckSummary.Total)
	}
}

func TestParsePRDetailJSON_StatusContextStates(t *testing.T) {
	// Test all StatusContext state mappings
	input := `{
		"number": 1,
		"title": "Test",
		"state": "OPEN",
		"isDraft": false,
		"mergeable": "MERGEABLE",
		"labels": [],
		"changedFiles": 0,
		"additions": 0,
		"deletions": 0,
		"reviewDecision": "",
		"reviews": [],
		"comments": [],
		"statusCheckRollup": [
			{"__typename": "StatusContext", "context": "ci/success", "state": "SUCCESS"},
			{"__typename": "StatusContext", "context": "ci/failure", "state": "FAILURE"},
			{"__typename": "StatusContext", "context": "ci/pending", "state": "PENDING"},
			{"__typename": "StatusContext", "context": "ci/error", "state": "ERROR"}
		],
		"url": "https://github.com/user/repo/pull/1"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil")
	}

	if len(pr.Checks) != 4 {
		t.Fatalf("Checks length = %d, want 4", len(pr.Checks))
	}

	// SUCCESS -> CheckConclusionSuccess
	if pr.Checks[0].Name != "ci/success" {
		t.Errorf("Checks[0].Name = %q, want %q", pr.Checks[0].Name, "ci/success")
	}
	if pr.Checks[0].Conclusion != CheckConclusionSuccess {
		t.Errorf("Checks[0].Conclusion = %q, want %q", pr.Checks[0].Conclusion, CheckConclusionSuccess)
	}

	// FAILURE -> CheckConclusionFailure
	if pr.Checks[1].Conclusion != CheckConclusionFailure {
		t.Errorf("Checks[1].Conclusion = %q, want %q", pr.Checks[1].Conclusion, CheckConclusionFailure)
	}

	// PENDING -> CheckStatusPending
	if pr.Checks[2].Conclusion != CheckStatusPending {
		t.Errorf("Checks[2].Conclusion = %q, want %q", pr.Checks[2].Conclusion, CheckStatusPending)
	}

	// ERROR -> CheckConclusionFailure
	if pr.Checks[3].Conclusion != CheckConclusionFailure {
		t.Errorf("Checks[3].Conclusion = %q, want %q", pr.Checks[3].Conclusion, CheckConclusionFailure)
	}

	// All StatusContext checks get Status = COMPLETED
	for i, c := range pr.Checks {
		if c.Status != CheckStatusCompleted {
			t.Errorf("Checks[%d].Status = %q, want %q (StatusContext always maps to COMPLETED)", i, c.Status, CheckStatusCompleted)
		}
	}

	// Summary: 1 passed (success), 2 failed (failure + error), 1 pending (pending mapped to PENDING conclusion)
	// Wait - PENDING conclusion is not SUCCESS, FAILURE, CANCELLED, or TIMED_OUT, so it falls to default = pending
	// ERROR maps to FAILURE conclusion with COMPLETED status, so it's a failure
	// So: passed=1, failed=2 (FAILURE + ERROR), pending=1 (PENDING)
	if pr.CheckSummary.Passed != 1 {
		t.Errorf("CheckSummary.Passed = %d, want 1", pr.CheckSummary.Passed)
	}
	if pr.CheckSummary.Failed != 2 {
		t.Errorf("CheckSummary.Failed = %d, want 2", pr.CheckSummary.Failed)
	}
	if pr.CheckSummary.Pending != 1 {
		t.Errorf("CheckSummary.Pending = %d, want 1", pr.CheckSummary.Pending)
	}
}

func TestParsePRDetailJSON_CheckRunConclusions(t *testing.T) {
	// Test CheckRun with various conclusions
	input := `{
		"number": 1,
		"title": "Test",
		"state": "OPEN",
		"isDraft": false,
		"mergeable": "MERGEABLE",
		"labels": [],
		"changedFiles": 0,
		"additions": 0,
		"deletions": 0,
		"reviewDecision": "",
		"reviews": [],
		"comments": [],
		"statusCheckRollup": [
			{"__typename": "CheckRun", "name": "build", "status": "COMPLETED", "conclusion": "SUCCESS"},
			{"__typename": "CheckRun", "name": "lint", "status": "COMPLETED", "conclusion": "FAILURE"},
			{"__typename": "CheckRun", "name": "deploy", "status": "COMPLETED", "conclusion": "CANCELLED"},
			{"__typename": "CheckRun", "name": "timeout", "status": "COMPLETED", "conclusion": "TIMED_OUT"},
			{"__typename": "CheckRun", "name": "neutral", "status": "COMPLETED", "conclusion": "NEUTRAL"},
			{"__typename": "CheckRun", "name": "action", "status": "COMPLETED", "conclusion": "ACTION_REQUIRED"},
			{"__typename": "CheckRun", "name": "queued", "status": "QUEUED", "conclusion": ""},
			{"__typename": "CheckRun", "name": "running", "status": "IN_PROGRESS", "conclusion": ""}
		],
		"url": "https://github.com/user/repo/pull/1"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil")
	}

	if len(pr.Checks) != 8 {
		t.Fatalf("Checks length = %d, want 8", len(pr.Checks))
	}

	// Summary: passed=1 (SUCCESS), failed=3 (FAILURE, CANCELLED, TIMED_OUT), pending=4 (NEUTRAL, ACTION_REQUIRED, QUEUED, IN_PROGRESS)
	if pr.CheckSummary.Total != 8 {
		t.Errorf("CheckSummary.Total = %d, want 8", pr.CheckSummary.Total)
	}
	if pr.CheckSummary.Passed != 1 {
		t.Errorf("CheckSummary.Passed = %d, want 1", pr.CheckSummary.Passed)
	}
	if pr.CheckSummary.Failed != 3 {
		t.Errorf("CheckSummary.Failed = %d, want 3", pr.CheckSummary.Failed)
	}
	if pr.CheckSummary.Pending != 4 {
		t.Errorf("CheckSummary.Pending = %d, want 4", pr.CheckSummary.Pending)
	}
}

func TestParsePRDetailJSON_ReviewsWithBodies(t *testing.T) {
	// Test that reviews with bodies count toward comment count
	input := `{
		"number": 5,
		"title": "Test",
		"state": "OPEN",
		"isDraft": false,
		"mergeable": "MERGEABLE",
		"labels": [],
		"changedFiles": 1,
		"additions": 1,
		"deletions": 0,
		"reviewDecision": "CHANGES_REQUESTED",
		"reviews": [
			{"author": {"login": "alice"}, "state": "CHANGES_REQUESTED", "body": "Please fix the tests"},
			{"author": {"login": "bob"}, "state": "COMMENTED", "body": ""},
			{"author": {"login": "charlie"}, "state": "APPROVED", "body": "Looks good now"}
		],
		"comments": [
			{"body": "general comment"}
		],
		"statusCheckRollup": [],
		"url": "https://github.com/user/repo/pull/5"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil")
	}

	// Reviewers
	if len(pr.Reviewers) != 3 {
		t.Fatalf("Reviewers length = %d, want 3", len(pr.Reviewers))
	}
	if pr.Reviewers[0].Login != "alice" || pr.Reviewers[0].State != "CHANGES_REQUESTED" {
		t.Errorf("Reviewers[0] = %+v, want alice/CHANGES_REQUESTED", pr.Reviewers[0])
	}
	if pr.Reviewers[1].Login != "bob" || pr.Reviewers[1].State != "COMMENTED" {
		t.Errorf("Reviewers[1] = %+v, want bob/COMMENTED", pr.Reviewers[1])
	}
	if pr.Reviewers[2].Login != "charlie" || pr.Reviewers[2].State != "APPROVED" {
		t.Errorf("Reviewers[2] = %+v, want charlie/APPROVED", pr.Reviewers[2])
	}

	// Comments: 1 general comment + 2 reviews with body (alice, charlie) = 3
	if pr.Comments != 3 {
		t.Errorf("Comments = %d, want 3 (1 general + 2 reviews with body)", pr.Comments)
	}

	if pr.ReviewStatus != ReviewChangesRequired {
		t.Errorf("ReviewStatus = %q, want %q", pr.ReviewStatus, ReviewChangesRequired)
	}
}

func TestParsePRDetailJSON_InvalidJSON(t *testing.T) {
	pr := parsePRDetailJSON([]byte(`not valid json`))
	if pr != nil {
		t.Error("parsePRDetailJSON should return nil for invalid JSON")
	}
}

func TestParsePRDetailJSON_EmptyInput(t *testing.T) {
	pr := parsePRDetailJSON([]byte(``))
	if pr != nil {
		t.Error("parsePRDetailJSON should return nil for empty input")
	}
}

func TestParsePRDetailJSON_NullFields(t *testing.T) {
	// Test with null/missing optional fields
	input := `{
		"number": 1,
		"title": "Minimal PR",
		"state": "OPEN",
		"isDraft": false,
		"mergeable": "UNKNOWN",
		"labels": null,
		"changedFiles": 0,
		"additions": 0,
		"deletions": 0,
		"reviewDecision": "",
		"reviews": null,
		"comments": null,
		"statusCheckRollup": null,
		"url": "https://github.com/user/repo/pull/1"
	}`

	pr := parsePRDetailJSON([]byte(input))
	if pr == nil {
		t.Fatal("parsePRDetailJSON returned nil for input with null fields")
	}

	if pr.Number != 1 {
		t.Errorf("Number = %d, want 1", pr.Number)
	}
	if len(pr.Labels) != 0 {
		t.Errorf("Labels length = %d, want 0", len(pr.Labels))
	}
	if len(pr.Reviewers) != 0 {
		t.Errorf("Reviewers length = %d, want 0", len(pr.Reviewers))
	}
	if pr.Comments != 0 {
		t.Errorf("Comments = %d, want 0", pr.Comments)
	}
	if len(pr.Checks) != 0 {
		t.Errorf("Checks length = %d, want 0", len(pr.Checks))
	}
	if pr.CheckSummary.Total != 0 {
		t.Errorf("CheckSummary.Total = %d, want 0", pr.CheckSummary.Total)
	}
}

func TestMapStatusContextState(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{"SUCCESS", CheckConclusionSuccess},
		{"FAILURE", CheckConclusionFailure},
		{"PENDING", CheckStatusPending},
		{"ERROR", CheckConclusionFailure},
		{"UNKNOWN_STATE", "UNKNOWN_STATE"},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := mapStatusContextState(tt.state)
			if got != tt.want {
				t.Errorf("mapStatusContextState(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestComputeCheckSummary(t *testing.T) {
	tests := []struct {
		name    string
		checks  []Check
		want    CheckSummary
	}{
		{
			"empty",
			nil,
			CheckSummary{Total: 0},
		},
		{
			"all passed",
			[]Check{
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionSuccess},
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionSuccess},
			},
			CheckSummary{Total: 2, Passed: 2},
		},
		{
			"mixed",
			[]Check{
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionSuccess},
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionFailure},
				{Status: CheckStatusInProgress, Conclusion: ""},
			},
			CheckSummary{Total: 3, Passed: 1, Failed: 1, Pending: 1},
		},
		{
			"cancelled and timed out are failures",
			[]Check{
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionCancelled},
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionTimedOut},
			},
			CheckSummary{Total: 2, Failed: 2},
		},
		{
			"neutral and action_required are pending",
			[]Check{
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionNeutral},
				{Status: CheckStatusCompleted, Conclusion: CheckConclusionActionRequired},
			},
			CheckSummary{Total: 2, Pending: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCheckSummary(tt.checks)
			if got != tt.want {
				t.Errorf("computeCheckSummary() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
