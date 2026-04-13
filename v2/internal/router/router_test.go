package router

import (
	"testing"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/model"
)

func TestPickBest(t *testing.T) {
	now := time.Now().UTC()
	decision := PickBest(Input{
		Now: now,
		Profiles: []model.Profile{
			{ID: "a", Provider: "openai", Frontend: "codex", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, Priority: 1},
			{ID: "b", Provider: "google", Frontend: "gemini_cli", Protocol: "native_cli", AuthMethod: "api_key", Enabled: true, Priority: 4},
		},
		Health: map[string]model.HealthSnapshot{
			"a": {RemainingRequests5Min: 15, RemainingRequestsHour: 100, EstimatedLatencyMS: 300, RecentErrorRatePercent: 1},
			"b": {RemainingRequests5Min: 45, RemainingRequestsHour: 500, EstimatedLatencyMS: 90, RecentErrorRatePercent: 0.1},
		},
		Request: model.TaskRequest{Frontend: "codex"},
	})
	if decision.ProfileID != "a" {
		t.Fatalf("expected a (frontend-scoped), got %s, reasons=%v rejected=%v", decision.ProfileID, decision.Reasons, decision.Rejected)
	}
}

func TestCooldownRejected(t *testing.T) {
	now := time.Now().UTC()
	d := PickBest(Input{
		Now: now,
		Profiles: []model.Profile{
			{ID: "a", Provider: "openai", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, CooldownUntil: now.Add(10 * time.Minute)},
		},
		Health:  map[string]model.HealthSnapshot{"a": {RemainingRequests5Min: 99}},
		Request: model.TaskRequest{},
	})
	if d.ProfileID != "" {
		t.Fatalf("expected none, got %s", d.ProfileID)
	}
	if len(d.Rejected) == 0 {
		t.Fatalf("expected rejection reasons")
	}
}

func TestRankReturnsFallbackCandidates(t *testing.T) {
	now := time.Now().UTC()
	candidates, rejected := Rank(Input{
		Now: now,
		Profiles: []model.Profile{
			{ID: "codex-main", Provider: "openai", Frontend: "codex", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, Priority: 10},
			{ID: "codex-backup", Provider: "openai", Frontend: "codex", Protocol: "app_server", AuthMethod: "chatgpt", Enabled: true, Priority: 4},
			{ID: "gemini-main", Provider: "google", Frontend: "gemini_cli", Protocol: "native_cli", AuthMethod: "google_login", Enabled: true, Priority: 8},
		},
		Health: map[string]model.HealthSnapshot{
			"codex-main":   {RemainingRequests5Min: 40, RemainingRequestsHour: 300, EstimatedLatencyMS: 120, RecentErrorRatePercent: 0.2},
			"codex-backup": {RemainingRequests5Min: 35, RemainingRequestsHour: 290, EstimatedLatencyMS: 140, RecentErrorRatePercent: 0.4},
			"gemini-main":  {RemainingRequests5Min: 60, RemainingRequestsHour: 500, EstimatedLatencyMS: 80, RecentErrorRatePercent: 0.1},
		},
		Request: model.TaskRequest{Frontend: "codex"},
	})
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates for codex frontend, got %d", len(candidates))
	}
	if candidates[0].ProfileID != "codex-main" {
		t.Fatalf("expected codex-main first, got %s", candidates[0].ProfileID)
	}
	if candidates[1].ProfileID != "codex-backup" {
		t.Fatalf("expected codex-backup second, got %s", candidates[1].ProfileID)
	}
	if len(rejected) == 0 {
		t.Fatalf("expected rejected entries for non-matching frontends")
	}
}
