package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stwalsh4118/navi/internal/metrics"
)

func TestSortSessions(t *testing.T) {
	sessions := []Info{
		{TmuxSession: "working", Status: StatusWorking, Timestamp: 100},
		{TmuxSession: "waiting", Status: StatusWaiting, Timestamp: 50},
		{TmuxSession: "permission", Status: StatusPermission, Timestamp: 75},
		{TmuxSession: "old-working", Status: StatusWorking, Timestamp: 25},
	}

	SortSessions(sessions)

	// Priority sessions first (waiting, permission)
	if sessions[0].Status != StatusWaiting && sessions[0].Status != StatusPermission {
		t.Errorf("Expected priority session first, got %q", sessions[0].Status)
	}

	// Non-priority sorted by timestamp descending
	lastNonPriority := sessions[len(sessions)-1]
	if lastNonPriority.Timestamp != 25 {
		t.Errorf("Expected oldest non-priority last, got timestamp %d", lastNonPriority.Timestamp)
	}
}

func TestAggregateMetrics(t *testing.T) {
	t.Run("empty sessions returns nil", func(t *testing.T) {
		result := AggregateMetrics([]Info{})
		if result != nil {
			t.Errorf("AggregateMetrics([]) should return nil, got %+v", result)
		}
	})

	t.Run("sessions without metrics returns nil", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "test1", Metrics: nil},
			{TmuxSession: "test2", Metrics: nil},
		}
		result := AggregateMetrics(sessions)
		if result != nil {
			t.Errorf("AggregateMetrics with no metrics should return nil, got %+v", result)
		}
	})

	t.Run("aggregates token metrics correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Tokens: &metrics.TokenMetrics{Input: 10000, Output: 5000, Total: 15000},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Tokens: &metrics.TokenMetrics{Input: 20000, Output: 8000, Total: 28000},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tokens.Input != 30000 {
			t.Errorf("Input = %d, want 30000", result.Tokens.Input)
		}
		if result.Tokens.Output != 13000 {
			t.Errorf("Output = %d, want 13000", result.Tokens.Output)
		}
		if result.Tokens.Total != 43000 {
			t.Errorf("Total = %d, want 43000", result.Tokens.Total)
		}
	})

	t.Run("aggregates time metrics correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Time: &metrics.TimeMetrics{TotalSeconds: 3600, WorkingSeconds: 2400, WaitingSeconds: 1200},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Time: &metrics.TimeMetrics{TotalSeconds: 1800, WorkingSeconds: 1200, WaitingSeconds: 600},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Time.TotalSeconds != 5400 {
			t.Errorf("TotalSeconds = %d, want 5400", result.Time.TotalSeconds)
		}
	})

	t.Run("aggregates tool counts correctly", func(t *testing.T) {
		sessions := []Info{
			{
				TmuxSession: "test1",
				Metrics: &metrics.Metrics{
					Tools: &metrics.ToolMetrics{Counts: map[string]int{"Read": 10, "Edit": 5}},
				},
			},
			{
				TmuxSession: "test2",
				Metrics: &metrics.Metrics{
					Tools: &metrics.ToolMetrics{Counts: map[string]int{"Read": 8, "Bash": 3}},
				},
			},
		}
		result := AggregateMetrics(sessions)
		if result == nil {
			t.Fatal("AggregateMetrics should not return nil")
		}
		if result.Tools.Counts["Read"] != 18 {
			t.Errorf("Read count = %d, want 18", result.Tools.Counts["Read"])
		}
	})
}

func TestHasPriorityTeammate(t *testing.T) {
	t.Run("returns false when Team is nil", func(t *testing.T) {
		s := Info{Team: nil}
		if HasPriorityTeammate(s) {
			t.Error("expected false for nil Team")
		}
	})

	t.Run("returns false when Agents is empty", func(t *testing.T) {
		s := Info{Team: &TeamInfo{Name: "test", Agents: []AgentInfo{}}}
		if HasPriorityTeammate(s) {
			t.Error("expected false for empty Agents")
		}
	})

	t.Run("returns false when all agents working or idle", func(t *testing.T) {
		s := Info{Team: &TeamInfo{
			Name: "test",
			Agents: []AgentInfo{
				{Name: "a", Status: StatusWorking},
				{Name: "b", Status: StatusIdle},
			},
		}}
		if HasPriorityTeammate(s) {
			t.Error("expected false when no priority agents")
		}
	})

	t.Run("returns true when agent has waiting status", func(t *testing.T) {
		s := Info{Team: &TeamInfo{
			Name: "test",
			Agents: []AgentInfo{
				{Name: "a", Status: StatusWorking},
				{Name: "b", Status: StatusWaiting},
			},
		}}
		if !HasPriorityTeammate(s) {
			t.Error("expected true for waiting agent")
		}
	})

	t.Run("returns true when agent has permission status", func(t *testing.T) {
		s := Info{Team: &TeamInfo{
			Name: "test",
			Agents: []AgentInfo{
				{Name: "a", Status: StatusPermission},
			},
		}}
		if !HasPriorityTeammate(s) {
			t.Error("expected true for permission agent")
		}
	})
}

func TestHasPriorityExternalAgent(t *testing.T) {
	t.Run("returns false when Agents is nil", func(t *testing.T) {
		s := Info{Agents: nil}
		if HasPriorityExternalAgent(s) {
			t.Error("expected false for nil Agents")
		}
	})

	t.Run("returns false when Agents is empty", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{}}
		if HasPriorityExternalAgent(s) {
			t.Error("expected false for empty Agents")
		}
	})

	t.Run("returns false when all external agents are idle", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{
			"opencode": {Status: StatusIdle},
			"other":    {Status: StatusStopped},
		}}
		if HasPriorityExternalAgent(s) {
			t.Error("expected false when no priority external agents")
		}
	})

	t.Run("returns false when external agent is working", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{
			"opencode": {Status: StatusWorking},
		}}
		if HasPriorityExternalAgent(s) {
			t.Error("expected false for working external agent")
		}
	})

	t.Run("returns true when external agent has permission status", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{
			"opencode": {Status: StatusPermission},
		}}
		if !HasPriorityExternalAgent(s) {
			t.Error("expected true for permission external agent")
		}
	})

	t.Run("returns true when external agent has waiting status", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{
			"opencode": {Status: StatusWaiting},
		}}
		if !HasPriorityExternalAgent(s) {
			t.Error("expected true for waiting external agent")
		}
	})

	t.Run("returns true when any external agent is priority", func(t *testing.T) {
		s := Info{Agents: map[string]ExternalAgent{
			"opencode": {Status: StatusIdle},
			"other":    {Status: StatusPermission},
		}}
		if !HasPriorityExternalAgent(s) {
			t.Error("expected true when one external agent has priority status")
		}
	})
}

func TestSortSessionsWithTeamPriority(t *testing.T) {
	t.Run("session with teammate permission sorts above plain working", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "plain-working", Status: StatusWorking, Timestamp: 100},
			{TmuxSession: "team-permission", Status: StatusWorking, Timestamp: 50, Team: &TeamInfo{
				Name:   "proj",
				Agents: []AgentInfo{{Name: "a", Status: StatusPermission}},
			}},
		}
		SortSessions(sessions)
		if sessions[0].TmuxSession != "team-permission" {
			t.Errorf("expected team-permission first, got %s", sessions[0].TmuxSession)
		}
	})

	t.Run("session with teammate waiting sorts with own waiting", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "plain-working", Status: StatusWorking, Timestamp: 200},
			{TmuxSession: "own-waiting", Status: StatusWaiting, Timestamp: 100},
			{TmuxSession: "team-waiting", Status: StatusWorking, Timestamp: 50, Team: &TeamInfo{
				Name:   "proj",
				Agents: []AgentInfo{{Name: "a", Status: StatusWaiting}},
			}},
		}
		SortSessions(sessions)
		// Both priority sessions should be before plain-working
		if sessions[2].TmuxSession != "plain-working" {
			t.Errorf("expected plain-working last, got %s", sessions[2].TmuxSession)
		}
	})

	t.Run("sessions without team sort as before", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "a", Status: StatusWorking, Timestamp: 100},
			{TmuxSession: "b", Status: StatusWaiting, Timestamp: 50},
			{TmuxSession: "c", Status: StatusWorking, Timestamp: 25},
		}
		SortSessions(sessions)
		if sessions[0].TmuxSession != "b" {
			t.Errorf("expected waiting first, got %s", sessions[0].TmuxSession)
		}
		if sessions[2].TmuxSession != "c" {
			t.Errorf("expected oldest working last, got %s", sessions[2].TmuxSession)
		}
	})
}

func TestSortSessionsWithExternalAgentPriority(t *testing.T) {
	t.Run("session with external permission sorts above plain working", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "plain-working", Status: StatusWorking, Timestamp: 100},
			{TmuxSession: "external-permission", Status: StatusWorking, Timestamp: 50, Agents: map[string]ExternalAgent{
				"opencode": {Status: StatusPermission},
			}},
		}
		SortSessions(sessions)
		if sessions[0].TmuxSession != "external-permission" {
			t.Errorf("expected external-permission first, got %s", sessions[0].TmuxSession)
		}
	})

	t.Run("session with external waiting sorts with own waiting", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "plain-working", Status: StatusWorking, Timestamp: 200},
			{TmuxSession: "own-waiting", Status: StatusWaiting, Timestamp: 100},
			{TmuxSession: "external-waiting", Status: StatusWorking, Timestamp: 50, Agents: map[string]ExternalAgent{
				"opencode": {Status: StatusWaiting},
			}},
		}
		SortSessions(sessions)
		if sessions[2].TmuxSession != "plain-working" {
			t.Errorf("expected plain-working last, got %s", sessions[2].TmuxSession)
		}
	})

	t.Run("sessions without external agents sort as before", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "a", Status: StatusWorking, Timestamp: 100},
			{TmuxSession: "b", Status: StatusWaiting, Timestamp: 50},
			{TmuxSession: "c", Status: StatusWorking, Timestamp: 25},
		}
		SortSessions(sessions)
		if sessions[0].TmuxSession != "b" {
			t.Errorf("expected waiting first, got %s", sessions[0].TmuxSession)
		}
		if sessions[2].TmuxSession != "c" {
			t.Errorf("expected oldest working last, got %s", sessions[2].TmuxSession)
		}
	})

	t.Run("combined team and external priority sessions sort before plain working", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "plain-working", Status: StatusWorking, Timestamp: 300},
			{TmuxSession: "team-priority", Status: StatusWorking, Timestamp: 200, Team: &TeamInfo{
				Name:   "proj",
				Agents: []AgentInfo{{Name: "a", Status: StatusPermission}},
			}},
			{TmuxSession: "external-priority", Status: StatusWorking, Timestamp: 100, Agents: map[string]ExternalAgent{
				"opencode": {Status: StatusWaiting},
			}},
		}
		SortSessions(sessions)
		if sessions[2].TmuxSession != "plain-working" {
			t.Errorf("expected plain-working last, got %s", sessions[2].TmuxSession)
		}
	})

	t.Run("session with external working does not sort as done", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "done-with-external-working", Status: "done", Timestamp: 50, Agents: map[string]ExternalAgent{
				"opencode": {Status: StatusWorking},
			}},
			{TmuxSession: "done-idle", Status: "done", Timestamp: 100},
		}
		SortSessions(sessions)
		if sessions[0].TmuxSession != "done-with-external-working" {
			t.Errorf("expected done-with-external-working first, got %s", sessions[0].TmuxSession)
		}
	})

	t.Run("waiting remains above external working regardless of recency", func(t *testing.T) {
		sessions := []Info{
			{TmuxSession: "external-working", Status: "done", Timestamp: 300, Agents: map[string]ExternalAgent{
				"opencode": {Status: StatusWorking},
			}},
			{TmuxSession: "waiting", Status: StatusWaiting, Timestamp: 100},
			{TmuxSession: "plain-done", Status: "done", Timestamp: 200},
		}

		SortSessions(sessions)

		if sessions[0].TmuxSession != "waiting" {
			t.Errorf("expected waiting first, got %s", sessions[0].TmuxSession)
		}
		if sessions[1].TmuxSession != "external-working" {
			t.Errorf("expected external-working second, got %s", sessions[1].TmuxSession)
		}
	})
}

func TestStatusStoppedConstant(t *testing.T) {
	if StatusStopped != "stopped" {
		t.Errorf("StatusStopped = %q, want %q", StatusStopped, "stopped")
	}
}

func TestHasPriorityTeammateIgnoresStoppedAgents(t *testing.T) {
	t.Run("returns false when all agents are stopped", func(t *testing.T) {
		s := Info{Team: &TeamInfo{
			Name: "test",
			Agents: []AgentInfo{
				{Name: "a", Status: StatusStopped},
				{Name: "b", Status: StatusStopped},
			},
		}}
		if HasPriorityTeammate(s) {
			t.Error("expected false when all agents are stopped")
		}
	})

	t.Run("returns true when stopped agent coexists with waiting agent", func(t *testing.T) {
		s := Info{Team: &TeamInfo{
			Name: "test",
			Agents: []AgentInfo{
				{Name: "a", Status: StatusStopped},
				{Name: "b", Status: StatusWaiting},
			},
		}}
		if !HasPriorityTeammate(s) {
			t.Error("expected true when a waiting agent exists alongside stopped")
		}
	})
}

func TestTeamInfoJSONRoundTrip(t *testing.T) {
	t.Run("Info with Team marshals and unmarshals correctly", func(t *testing.T) {
		// Test is just compilation + basic structure - JSON round-trip
		info := Info{
			TmuxSession: "test",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Team: &TeamInfo{
				Name: "my-team",
				Agents: []AgentInfo{
					{Name: "researcher", Status: StatusWorking, Timestamp: 1234567890},
					{Name: "implementer", Status: StatusIdle, Timestamp: 1234567880},
				},
			},
		}

		// Marshal
		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		// Verify team field is present
		if !strings.Contains(string(data), `"team"`) {
			t.Error("marshaled JSON should contain team field")
		}

		// Unmarshal
		var decoded Info
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded.Team == nil {
			t.Fatal("decoded Team should not be nil")
		}
		if decoded.Team.Name != "my-team" {
			t.Errorf("Team.Name = %q, want %q", decoded.Team.Name, "my-team")
		}
		if len(decoded.Team.Agents) != 2 {
			t.Fatalf("Team.Agents length = %d, want 2", len(decoded.Team.Agents))
		}
		if decoded.Team.Agents[0].Name != "researcher" {
			t.Errorf("Agent[0].Name = %q, want %q", decoded.Team.Agents[0].Name, "researcher")
		}
	})

	t.Run("Info without Team omits team from JSON", func(t *testing.T) {
		info := Info{
			TmuxSession: "test",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Team:        nil,
		}

		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		if strings.Contains(string(data), `"team"`) {
			t.Error("marshaled JSON should NOT contain team field when nil")
		}
	})
}

func TestExternalAgentJSONBehavior(t *testing.T) {
	t.Run("JSON round-trip with agents", func(t *testing.T) {
		info := Info{
			TmuxSession: "test-session",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Agents: map[string]ExternalAgent{
				"opencode": {
					Status:    StatusIdle,
					Timestamp: 1234567888,
				},
			},
		}

		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded Info
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		opencode, ok := decoded.Agents["opencode"]
		if !ok {
			t.Fatal("decoded Agents missing opencode entry")
		}
		if opencode.Status != StatusIdle {
			t.Errorf("opencode status = %q, want %q", opencode.Status, StatusIdle)
		}
		if opencode.Timestamp != 1234567888 {
			t.Errorf("opencode timestamp = %d, want %d", opencode.Timestamp, int64(1234567888))
		}
	})

	t.Run("omit agents when nil", func(t *testing.T) {
		info := Info{
			TmuxSession: "test-session",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Agents:      nil,
		}

		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		if strings.Contains(string(data), `"agents"`) {
			t.Error("marshaled JSON should NOT contain agents field when nil")
		}
	})

	t.Run("omit agents when empty", func(t *testing.T) {
		info := Info{
			TmuxSession: "test-session",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Agents:      map[string]ExternalAgent{},
		}

		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		if strings.Contains(string(data), `"agents"`) {
			t.Error("marshaled JSON should NOT contain agents field when empty")
		}
	})

	t.Run("backward compatibility without agents", func(t *testing.T) {
		const legacyJSON = `{"tmux_session":"legacy","status":"working","message":"ok","cwd":"/tmp","timestamp":123}`

		var decoded Info
		if err := json.Unmarshal([]byte(legacyJSON), &decoded); err != nil {
			t.Fatalf("Unmarshal failed for legacy JSON: %v", err)
		}

		if decoded.Agents != nil {
			t.Fatalf("Agents = %#v, want nil for legacy JSON", decoded.Agents)
		}
		if decoded.TmuxSession != "legacy" {
			t.Errorf("TmuxSession = %q, want %q", decoded.TmuxSession, "legacy")
		}
		if decoded.Status != StatusWorking {
			t.Errorf("Status = %q, want %q", decoded.Status, StatusWorking)
		}
	})

	t.Run("multiple agents round-trip", func(t *testing.T) {
		info := Info{
			TmuxSession: "test-session",
			Status:      StatusWorking,
			Timestamp:   1234567890,
			Agents: map[string]ExternalAgent{
				"claude": {
					Status:    StatusWorking,
					Timestamp: 1234567890,
				},
				"opencode": {
					Status:    StatusPermission,
					Timestamp: 1234567880,
				},
			},
		}

		data, err := json.Marshal(info)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded Info
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if len(decoded.Agents) != 2 {
			t.Fatalf("Agents length = %d, want 2", len(decoded.Agents))
		}
		if decoded.Agents["claude"].Status != StatusWorking {
			t.Errorf("claude status = %q, want %q", decoded.Agents["claude"].Status, StatusWorking)
		}
		if decoded.Agents["opencode"].Status != StatusPermission {
			t.Errorf("opencode status = %q, want %q", decoded.Agents["opencode"].Status, StatusPermission)
		}
	})
}
